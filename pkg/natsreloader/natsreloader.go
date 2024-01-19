// Copyright 2020-2023 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package natsreloader

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const errorFmt = "Error: %s\n"

// Config represents the configuration of the reloader.
type Config struct {
	PidFile       string
	WatchedFiles  []string
	MaxRetries    int
	RetryWaitSecs int
	Signal        os.Signal
}

// Reloader monitors the state from a single server config file
// and sends signal on updates.
type Reloader struct {
	*Config

	// proc represents the NATS Server process which will
	// be signaled.
	proc *os.Process

	// pid is the last known PID from the NATS Server.
	pid int

	// quit shutsdown the reloader.
	quit func()
}

func (r *Reloader) waitForProcess() error {
	var proc *os.Process
	var pid int
	attempts := 0

	startTime := time.Now()
	for {
		pidfile, err := os.ReadFile(r.PidFile)
		if err != nil {
			goto WaitAndRetry
		}

		pid, err = strconv.Atoi(string(pidfile))
		if err != nil {
			goto WaitAndRetry
		}

		// This always succeeds regardless of the process existing or not.
		proc, err = os.FindProcess(pid)
		if err != nil {
			goto WaitAndRetry
		}

		// Check if the process is still alive.
		err = proc.Signal(syscall.Signal(0))
		if err != nil {
			goto WaitAndRetry
		}
		break

	WaitAndRetry:
		log.Printf("Error while monitoring pid %v: %v", pid, err)
		attempts++
		if attempts > r.MaxRetries {
			return fmt.Errorf("too many errors attempting to find server process")
		}
		time.Sleep(time.Duration(r.RetryWaitSecs) * time.Second)
	}

	if attempts > 0 {
		log.Printf("Found pid from pidfile %q after %v failed attempts (took %.3fs)",
			r.PidFile, attempts, time.Since(startTime).Seconds())
	}
	r.proc = proc
	return nil
}

func removeDuplicateStrings(s []string) []string {
	if len(s) < 1 {
		return s
	}

	sort.Strings(s)
	prev := 1
	for curr := 1; curr < len(s); curr++ {
		if s[curr-1] != s[curr] {
			s[prev] = s[curr]
			prev++
		}
	}

	return s[:prev]
}

func getFileDigest(filePath string) ([]byte, error) {
	h := sha256.New()
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func handleEvent(event fsnotify.Event, lastConfigAppliedCache map[string][]byte, updatedFiles, deletedFiles []string) ([]string, []string) {
	if event.Has(fsnotify.Remove) {
		// We don't get a Remove event for the directory itself, so
		// we need to detect that separately.
		return updatedFiles, append(deletedFiles, event.Name)
	}
	_, err := os.Stat(event.Name)
	if err != nil {
		// Beware that this means that we won't reconfigure if a file
		// is permanently removed.  We want to support transient
		// disappearance, waiting for the new content, and have not set
		// up any sort of longer-term timers to detect permanent
		// deletion.
		// If you really need this, then switch a file to be empty
		// before removing if afterwards.
		return updatedFiles, deletedFiles
	}

	if len(updatedFiles) > 0 {
		return updatedFiles, deletedFiles
	}
	digest, err := getFileDigest(event.Name)
	if err != nil {
		log.Printf(errorFmt, err)
		return updatedFiles, deletedFiles
	}

	lastConfigHash, ok := lastConfigAppliedCache[event.Name]
	if ok && bytes.Equal(lastConfigHash, digest) {
		return updatedFiles, deletedFiles
	}

	log.Printf("Changed config; file=%q existing=%v total-files=%d",
		event.Name, ok, len(lastConfigAppliedCache))
	lastConfigAppliedCache[event.Name] = digest
	return append(updatedFiles, event.Name), deletedFiles
}

// handleEvents handles all events in the queue. It returns the updated and deleted files and can contain duplicates.
func handleEvents(configWatcher *fsnotify.Watcher, event fsnotify.Event, lastConfigAppliedCache map[string][]byte) ([]string, []string) {
	updatedFiles, deletedFiles := handleEvent(event, lastConfigAppliedCache, make([]string, 0, 16), make([]string, 0, 16))
	for {
		select {
		case event := <-configWatcher.Events:
			updatedFiles, deletedFiles = handleEvent(event, lastConfigAppliedCache, updatedFiles, deletedFiles)
		default:
			return updatedFiles, deletedFiles
		}
	}
}

func handleDeletedFiles(deletedFiles []string, configWatcher *fsnotify.Watcher, lastConfigAppliedCache map[string][]byte) ([]string, []string) {
	if len(deletedFiles) > 0 {
		log.Printf("Tracking files %v", deletedFiles)
	}
	newDeletedFiles := make([]string, 0, len(deletedFiles))
	updated := make([]string, 0, len(deletedFiles))
	for _, f := range deletedFiles {
		if err := configWatcher.Add(f); err != nil {
			newDeletedFiles = append(newDeletedFiles, f)
		} else {
			updated, _ = handleEvent(fsnotify.Event{Name: f, Op: fsnotify.Create}, lastConfigAppliedCache, updated, nil)
		}
	}
	return removeDuplicateStrings(updated), newDeletedFiles
}

func (r *Reloader) init() (*fsnotify.Watcher, map[string][]byte, error) {
	err := r.waitForProcess()
	if err != nil {
		return nil, nil, err
	}

	configWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	watchedFiles := make([]string, 0)

	for _, c := range r.WatchedFiles {
		childFiles, err := getServerFiles(c)
		if err != nil {
			return nil, nil, err
		}

		watchedFiles = append(watchedFiles, childFiles...)
	}

	r.WatchedFiles = append(r.WatchedFiles, watchedFiles...)

	// Follow configuration updates in the directory where
	// the config file is located and trigger reload when
	// it is either recreated or written into.
	for i := range r.WatchedFiles {
		// Ensure our paths are canonical
		r.WatchedFiles[i], _ = filepath.Abs(r.WatchedFiles[i])
	}
	r.WatchedFiles = removeDuplicateStrings(r.WatchedFiles)
	// Follow configuration file updates and trigger reload when
	// it is either recreated or written into.
	for i := range r.WatchedFiles {
		// Watch files individually for https://github.com/kubernetes/kubernetes/issues/112677
		if err := configWatcher.Add(r.WatchedFiles[i]); err != nil {
			_ = configWatcher.Close()
			return nil, nil, err
		}
		log.Printf("Watching file: %v", r.WatchedFiles[i])
	}

	// lastConfigAppliedCache is the last config update
	// applied by us.
	lastConfigAppliedCache := make(map[string][]byte)

	// Preload config hashes, so we know their digests
	// up front and avoid potentially reloading when unnecessary.
	for _, configFile := range r.WatchedFiles {
		digest, err := getFileDigest(configFile)
		if err != nil {
			_ = configWatcher.Close()
			return nil, nil, err
		}
		lastConfigAppliedCache[configFile] = digest
	}
	log.Printf("Live, ready to kick pid %v on config changes (files=%d)",
		r.proc.Pid, len(lastConfigAppliedCache))

	if len(lastConfigAppliedCache) == 0 {
		log.Printf("Error: no watched config files cached; input spec was: %#v",
			r.WatchedFiles)
	}
	return configWatcher, lastConfigAppliedCache, nil
}

func (r *Reloader) reload(updatedFiles []string) error {
	attempts := 0
	for {
		err := r.waitForProcess()
		if err != nil {
			goto Retry
		}

		log.Printf("Sending pid %v '%s' signal to reload changes from: %s", r.proc.Pid, r.Signal.String(), updatedFiles)
		err = r.proc.Signal(r.Signal)
		if err == nil {
			return nil
		}

	Retry:
		if err != nil {
			log.Printf("Error during reload: %s", err)
		}
		if attempts > r.MaxRetries {
			return fmt.Errorf("too many errors (%v) attempting to signal server to reload: %w", attempts, err)
		}
		delay := retryJitter(time.Duration(r.RetryWaitSecs) * time.Second)
		log.Printf("Wait and retrying in %.3fs ...", delay.Seconds())
		time.Sleep(delay)
		attempts++
	}
}

// Run starts the main loop.
func (r *Reloader) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	r.quit = func() {
		cancel()
	}

	configWatcher, lastConfigAppliedCache, err := r.init()
	if err != nil {
		return err
	}
	defer configWatcher.Close()

	// We use a ticker to re-add deleted files to the watcher
	t := time.NewTicker(time.Second)
	t.Stop()
	defer t.Stop()
	var tickerRunning bool
	var deletedFiles []string
	var updatedFiles []string

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			updatedFiles, deletedFiles = handleDeletedFiles(deletedFiles, configWatcher, lastConfigAppliedCache)
			if len(deletedFiles) == 0 {
				log.Printf("All monitored files detected.")
				t.Stop()
				tickerRunning = false
			}
			if len(updatedFiles) > 0 {
				// Send signal to reload the config.
				log.Printf("Updated files: %v", updatedFiles)
				break
			}
			continue
		case event := <-configWatcher.Events:
			updated, deleted := handleEvents(configWatcher, event, lastConfigAppliedCache)
			updatedFiles = removeDuplicateStrings(updated)
			deletedFiles = removeDuplicateStrings(append(deletedFiles, deleted...))
			if !tickerRunning {
				// Start the ticker to re-add deleted files.
				log.Printf("Starting ticker to re-add all tracked files.")
				t.Reset(time.Second)
				tickerRunning = true
			}
			if len(updatedFiles) > 0 {
				// Send signal to reload the config
				log.Printf("Updated files: %v", updatedFiles)
				break
			}
			continue
		case err := <-configWatcher.Errors:
			log.Printf(errorFmt, err)
			continue
		}
		// Configuration was updated, try to do reload for a few times
		// otherwise give up and wait for next event.
		err := r.reload(updatedFiles)
		if err != nil {
			return err
		}
		updatedFiles = nil
	}
}

// Stop shutsdown the process.
func (r *Reloader) Stop() error {
	log.Println("Shutting down...")
	r.quit()
	return nil
}

// NewReloader returns a configured NATS server reloader.
func NewReloader(config *Config) (*Reloader, error) {
	return &Reloader{
		Config: config,
	}, nil
}

// retryJitter helps avoid trying things at synchronized times, thus improving
// resiliency in aggregate.
func retryJitter(base time.Duration) time.Duration {
	b := float64(base)
	// 10% +/-
	offset := rand.Float64()*0.2 - 0.1
	return time.Duration(b + offset)
}

func getServerFiles(configFile string) ([]string, error) {
	filePaths, err := getIncludePaths(configFile, make(map[string]interface{}))
	if err != nil {
		return nil, err
	}

	certPaths, err := getCertPaths(filePaths)
	if err != nil {
		return nil, err
	}

	filePaths = append(filePaths, certPaths...)
	sort.Strings(filePaths)

	return filePaths, nil
}

func getIncludePaths(configFile string, checked map[string]interface{}) ([]string, error) {
	if _, ok := checked[configFile]; ok {
		return []string{}, nil
	}

	configFile, err := filepath.Abs(configFile)
	if err != nil {
		return nil, err
	}

	// Add the current config file to the list and mark it as checked
	filePaths := []string{configFile}
	checked[configFile] = nil

	parentDirectory := filepath.Dir(configFile)
	includeRegex := regexp.MustCompile(`(?m)^\s*include\s+([^\n]*)`)

	content, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	includeMatches := includeRegex.FindAllStringSubmatch(string(content), -1)
	for _, match := range includeMatches {
		// Include filepaths in NATS config are always relative
		fullyQualifiedPath := filepath.Join(parentDirectory, match[1])
		fullyQualifiedPath = filepath.Clean(fullyQualifiedPath)

		if _, err := os.Stat(fullyQualifiedPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("%s does not exist", fullyQualifiedPath)
		}

		// Recursive call to make sure we catch any nested includes
		// Using map[string]interface{} as a set to avoid loops
		includePaths, err := getIncludePaths(fullyQualifiedPath, checked)
		if err != nil {
			return nil, err
		}

		filePaths = append(filePaths, includePaths...)
	}

	return filePaths, nil
}

func getCertPaths(configPaths []string) ([]string, error) {
	certPaths := []string{}
	certRegex := regexp.MustCompile(`(?m)^\s*(cert_file|key_file|ca_file)\s*:\s*"?([^"\n]*)"?`)

	for _, configPath := range configPaths {
		content, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		certMatches := certRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range certMatches {
			fullyQualifiedPath, err := filepath.Abs(match[2])
			if err != nil {
				return nil, err
			}

			if _, err := os.Stat(fullyQualifiedPath); os.IsNotExist(err) {
				return nil, fmt.Errorf("%s does not exist", fullyQualifiedPath)
			}

			certPaths = append(certPaths, fullyQualifiedPath)
		}
	}

	return certPaths, nil
}
