package natsreloadertest

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/nats-io/nack/pkg/natsreloader"
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
	nconfig := &natsreloader.Config{
		PidFile:     pidfile.Name(),
		ConfigFiles: []string{},
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
		nconfig.ConfigFiles = append(nconfig.ConfigFiles, configFile.Name())
	}

	r, err := natsreloader.NewReloader(nconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	var signals = 0

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
		time.Sleep(100 * time.Millisecond)
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
	if err != nil && err != context.Canceled {
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
