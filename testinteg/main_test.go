package testinteg

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func runToolStdoutPipe(w io.Writer, name string, args ...string) (io.ReadCloser, error) {
	cmd, err := toolCmd(name, args...)
	if err != nil {
		return nil, err
	}

	if sp, ok := w.(interface{ SetPrefix(string) }); ok {
		sp.SetPrefix(cmd.String())
	}
	cmd.Stderr = w

	rc, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		rc.Close()
		return nil, err
	}

	return rc, nil
}

func runTool(w io.Writer, name string, args ...string) error {
	cmd, err := toolCmd(name, args...)
	if err != nil {
		return err
	}

	if lfw, ok := w.(*logfWriter); ok {
		w = &logfWriter{logger: lfw.logger, prefix: cmd.String()}
	}
	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s: %w", cmd.String(), err)
	}
	return nil
}

func toolCmd(name string, args ...string) (*exec.Cmd, error) {
	toolDir := os.Getenv("TOOLDIR")
	if toolDir == "" {
		return nil, fmt.Errorf("failed to create %q cmd, missing TOOLDIR", name)
	}

	cmd := exec.Command(filepath.Join(toolDir, name), args...)
	cmd.Env = []string{
		fmt.Sprintf("KUBECONFIG=%s", filepath.Join(toolDir, "kubeconfig.yaml")),
	}

	return cmd, nil
}

type logfWriter struct {
	prefix string
	logger interface {
		Logf(format string, args ...interface{})
	}
}

func (w *logfWriter) Write(p []byte) (n int, err error) {
	var pre string
	if w.prefix != "" {
		pre = fmt.Sprintf("%s: ", w.prefix)
	}

	w.logger.Logf("%s%s", pre, p)
	return len(p), nil
}

func retry(n int, d time.Duration, fn func() error) error {
	var err error
	for i := 0; i < n; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(d)
	}
	return fmt.Errorf("failed %d times: %w", n, err)
}
