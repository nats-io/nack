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
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const (
	testConfig_0 = `
jetstream {
	store_dir: data/jetstream
	max_mem: 10G
	max_file: 10G
}
operator: eyJ0eXAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ.eyJqdGkiOiI3Sk1aNEQ0RE1WU1hGWDRYWExCTVVITjY1MjdaQlhaV0dZUUtBQVk0TVRQQTZOSEdHS1NBIiwiaWF0IjoxNjgyNTAzMzg4LCJpc3MiOiJPQ1hXU0tWNU5UTUxCUjI0NlZaQ0laSzVZRlBVTVNPVjZVWk5JNDRPTFVVUUlCNkE0VU1RWE1USiIsIm5hbWUiOiJuYXRzLXRlc3QtMDEiLCJzdWIiOiJPQ1hXU0tWNU5UTUxCUjI0NlZaQ0laSzVZRlBVTVNPVjZVWk5JNDRPTFVVUUlCNkE0VU1RWE1USiIsIm5hdHMiOnsic2lnbmluZ19rZXlzIjpbIk9DS1oyNkpJUDdCN1BHQk43QTdEVEVHVk9NUlNHNE5XVFZURjdPQ0pOSVdRS0xZT0YzWDJBTlFKIl0sImFjY291bnRfc2VydmVyX3VybCI6Im5hdHM6Ly9sb2NhbGhvc3Q6NDIyMiIsIm9wZXJhdG9yX3NlcnZpY2VfdXJscyI6WyJuYXRzOi8vbG9jYWxob3N0OjQyMjIiXSwic3lzdGVtX2FjY291bnQiOiJBQk5ITEY2NVlEWkxGWUlIUVVVU0pXWlZSUVc0UE8zVFFRT0VTNlA3WTRUQ1BQWVVTNkhIVzJFUyIsInR5cGUiOiJvcGVyYXRvciIsInZlcnNpb24iOjJ9fQ.LjVkEnA3Fg3F20cPZm5FShZQKWPiU4pLdhh2s0cj_zhxA88wXgNfUo_SPs59JE97qvpR7AOWksP5dzxMZJ2iBQ
# System Account named SYS
system_account: ABNHLF65YDZLFYIHQUUSJWZVRQW4PO3TQQOES6P7Y4TCPPYUS6HHW2ES

resolver {
	type: full
	dir: './'
}

jetstream {
	store_dir: data/jetstream
	max_mem: 10G
	max_file: 10G
}

include './testConfig_1.conf'`

	testConfig_1 = `include ./testConfig_2.conf`

	testConfig_2 = `
tls: {
	cert_file: "./test.pem"
	key_file: "./testkey.pem"
}
`
	includeTest_0 = `
include nats_0.conf
include  nats_1.conf;	// semicolon terminated
include "nats_2.conf"	// double-quoted
include  "nats_3.conf"; // double-quoted and semicolon terminated
include 'nats_4.conf'	// single-quoted
include  'nats_5.conf'; // single-quoted and semicolon terminated
include $NATS;        	// ignore variable
include "$NATS_6.conf"  // filename starting with $
include includeTest_1.conf
`
	includeTest_1 = `
tls: {
	cert_file: ./nats_0.pem
	key_file: 'nats_0.key'
}
tls: {
	cert_file: "./nats_1.pem"
	key_file: $test
}
tls: {
	cert_file: "$nats_2.pem";
	key_file: 'nats_1.key';
}
`
)

var configContents = `port = 2222`
var newConfigContents = `port = 2222
someOtherThing = "bar"
`

func TestReloader(t *testing.T) {
	// Setup a pidfile that points to us
	pid := os.Getpid()
	pidfile, err := os.CreateTemp(os.TempDir(), "nats-pid-")
	if err != nil {
		t.Fatal(err)
	}

	p := fmt.Sprintf("%d", pid)
	if _, err := pidfile.WriteString(p); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pidfile.Name())

	// Create tempfile with contents, then update it
	nconfig := &Config{
		PidFile:      pidfile.Name(),
		WatchedFiles: []string{},
		Signal:       syscall.SIGHUP,
	}

	var configFiles []*os.File
	for i := 0; i < 2; i++ {
		configFile, err := os.CreateTemp(os.TempDir(), "nats-conf-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(configFile.Name())

		if _, err := configFile.WriteString(configContents); err != nil {
			t.Fatal(err)
		}
		configFiles = append(configFiles, configFile)
		nconfig.WatchedFiles = append(nconfig.WatchedFiles, configFile.Name())
	}

	r, err := NewReloader(nconfig)
	if err != nil {
		t.Fatal(err)
	}

	signals := 0

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var sigsMu sync.Mutex

	// Signal handling.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)

		// Success when receiving the first signal
		for range c {
			sigsMu.Lock()
			signals++
			sigsMu.Unlock()
		}
	}()

	go func() {
		// This is terrible, but we need this thread to wait until r.Run(ctx) has finished starting up
		// before we start mucking with the file.
		// There isn't any other good way to synchronize on this happening.
		time.Sleep(200 * time.Millisecond)
		for _, configfile := range configFiles {
			for i := 0; i < 5; i++ {
				// Append some more stuff to the config
				if _, err := configfile.WriteAt([]byte(newConfigContents), 0); err != nil {
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Create some random file in the same directory, shouldn't trigger an
		// additional server signal.
		configFile, err := os.CreateTemp(os.TempDir(), "foo")
		if err != nil {
			t.Log(err)
			return
		}
		defer os.Remove(configFile.Name())
		time.Sleep(100 * time.Millisecond)

		cancel()
	}()

	err = r.Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	// We should have gotten only one signal for each configuration file
	sigsMu.Lock()
	got := signals
	sigsMu.Unlock()
	expected := len(configFiles)
	if got != expected {
		t.Fatalf("Wrong number of signals received. Expected: %v, got: %v", expected, got)
	}
}

func TestInclude(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	dummyFiles := []string{
		"nats_0.conf",
		"nats_1.conf",
		"nats_2.conf",
		"nats_3.conf",
		"nats_4.conf",
		"nats_5.conf",
		"$NATS_6.conf",
		"nats_0.pem",
		"nats_1.pem",
		"$nats_2.pem",
		"nats_0.key",
		"nats_1.key",
	}

	for _, f := range dummyFiles {
		p := filepath.Join(directory, f)
		err = writeFile("", p)
		defer os.Remove(p)
		if err != nil {
			t.Fatal(err)
		}
	}

	includeTestConf_0 := filepath.Join(directory, "includeTest_0.conf")
	err = writeFile(includeTest_0, includeTestConf_0)
	defer os.Remove(includeTestConf_0)
	if err != nil {
		t.Fatal(err)
	}

	includeTestConf_1 := filepath.Join(directory, "includeTest_1.conf")
	err = writeFile(includeTest_1, includeTestConf_1)
	defer os.Remove(includeTestConf_1)
	if err != nil {
		t.Fatal(err)
	}

	includes, err := getServerFiles("includeTest_0.conf")
	if err != nil {
		t.Fatal(err)
	}

	includePaths := make([]string, 0)
	for _, p := range includes {
		includePaths = append(includePaths, filepath.Base(p))
	}

	dummyFiles = append(dummyFiles, "includeTest_0.conf")
	dummyFiles = append(dummyFiles, "includeTest_1.conf")

	sort.Strings(dummyFiles)
	sort.Strings(includePaths)

	for i, p := range dummyFiles {
		if p != includePaths[i] {
			t.Fatal("Expected include paths do not match")
		}
	}

}

func TestFileFinder(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	confFile := filepath.Join(directory, "testConfig_0.conf")
	err = writeFile(testConfig_0, confFile)
	defer os.Remove(confFile)
	if err != nil {
		t.Fatal(err)
	}
	confFile = filepath.Join(directory, "testConfig_1.conf")
	err = writeFile(testConfig_1, confFile)
	defer os.Remove(confFile)
	if err != nil {
		t.Fatal(err)
	}
	confFile = filepath.Join(directory, "testConfig_2.conf")
	err = writeFile(testConfig_2, confFile)
	defer os.Remove(confFile)
	if err != nil {
		t.Fatal(err)
	}
	confFile = filepath.Join(directory, "test.pem")
	err = writeFile("test", confFile)
	defer os.Remove(confFile)
	if err != nil {
		t.Fatal(err)
	}
	confFile = filepath.Join(directory, "testkey.pem")
	err = writeFile("test", confFile)
	defer os.Remove(confFile)
	if err != nil {
		t.Fatal(err)
	}

	pid := os.Getpid()
	pidFile := filepath.Join(directory, "nats.pid")
	err = writeFile(fmt.Sprintf("%d", pid), pidFile)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pidFile)

	nconfig := &Config{
		PidFile:      pidFile,
		WatchedFiles: []string{filepath.Join(directory, "testConfig_0.conf")},
		Signal:       syscall.SIGHUP,
	}

	r, err := NewReloader(nconfig)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		err = r.Run(ctx)
		if err != nil {
			t.Error(err)
		}
	}()

	time.Sleep(time.Second)

	cancel()

	expectedWatchedFiles := []string{
		"/testConfig_0.conf",
		"/testConfig_1.conf",
		"/testConfig_2.conf",
		"/test.pem",
		"/testkey.pem",
	}

	watchedFiles := r.WatchedFiles

	sort.Strings(expectedWatchedFiles)
	sort.Strings(watchedFiles)

	if len(watchedFiles) > len(expectedWatchedFiles) {
		t.Fatal("Unexpected number of watched files")
	}

	for i, e := range expectedWatchedFiles {
		f := strings.TrimPrefix(watchedFiles[i], directory)
		if f != e {
			t.Fatal("Expected watched file list does not match")
		}

	}
}

func writeFile(content, path string) error {
	parentDirectory := filepath.Dir(path)
	if _, err := os.Stat(parentDirectory); errors.Is(err, fs.ErrNotExist) {
		err = os.MkdirAll(parentDirectory, 0o755)
		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}
