package testinteg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/nats-io/nats.go"
)

const (
	natsRepo = "https://nats-io.github.io/k8s/helm/charts"
)

func TestInstallStream(t *testing.T) {
	lw := &logfWriter{logger: t}

	if err := helmRepoAdd(lw, "nats", natsRepo); err != nil {
		t.Fatal(err)
	}
	defer errCheck(t, func() error { return helmRepoRemove(lw, "nats") })

	if err := installNATS(lw, "testdata/helm-nats.yaml"); err != nil {
		t.Fatal(err)
	}
	defer errCheck(t, func() error { return uninstallNATS(lw) })

	if err := kubectlApply(lw, "../deploy/crds.yml"); err != nil {
		t.Fatal(err)
	}
	defer errCheck(t, func() error { return kubectlDelete(lw, "../deploy/crds.yml") })

	if err := installNACK(lw, "testdata/helm-jetstream-controller.yaml"); err != nil {
		t.Fatal(err)
	}
	defer errCheck(t, func() error { return uninstallNACK(lw) })

	streams := []string{
		"stream-00.yaml",
		"stream-01.yaml",
	}
	for _, stream := range streams {
		streamPath := filepath.Join("testdata", stream)
		streamName := strings.TrimSuffix(stream, ".yaml")

		func() {
			if err := installStream(lw, streamName, streamPath); err != nil {
				t.Fatalf("%s: %s", streamName, err)
			}
			defer errCheck(t, func() error { return uninstallStream(lw, streamPath) })

			streamConfig, err := getStreamConfig(lw, streamName)
			if err != nil {
				t.Fatalf("%s: %s", streamName, err)
			}

			streamSpec, err := getStreamSpec(lw, streamName)
			if err != nil {
				t.Fatalf("%s: %s", streamName, err)
			}

			if err := equalStreamConfigSpec(streamConfig, streamSpec); err != nil {
				t.Fatalf("%s: %s", streamName, err)
			}
		}()
	}
}

func equalStreamConfigSpec(a nats.StreamConfig, b v1beta2.StreamSpec) error {
	var msgs []string
	diff := func(name string, a, b interface{}) {
		msgs = append(msgs, fmt.Sprintf("%s(%+v != %+v)", name, a, b))
	}

	checkSource := func(am *nats.StreamSource, bm *v1beta2.StreamSource) {
		if am.Name != bm.Name {
			diff("Source.Name", am.Name, bm.Name)
		}
		if am.OptStartSeq != uint64(bm.OptStartSeq) {
			diff("Source.OptStartSeq", am.OptStartSeq, bm.OptStartSeq)
		}
		if got, want := am.OptStartTime != nil, bm.OptStartTime != ""; got != want {
			diff("Source.OptStartTime", am.OptStartTime, bm.OptStartTime)
		} else if got && want {
			at := am.OptStartTime
			bt, err := time.Parse("", bm.OptStartTime)
			if err != nil || !at.Equal(bt) {
				diff("Source.OptStartTime", am.OptStartTime, bm.OptStartTime)
			}
		}
		if am.FilterSubject != bm.FilterSubject {
			diff("Source.FilterSubject", am.FilterSubject, bm.FilterSubject)
		}
		if am.External != nil && am.External.APIPrefix != bm.ExternalAPIPrefix {
			diff("Source.External.APIPrefix", am.External.APIPrefix, bm.ExternalAPIPrefix)
		}
		if am.External != nil && am.External.DeliverPrefix != bm.ExternalDeliverPrefix {
			diff("Source.External.DeliverPrefix", am.External.DeliverPrefix, bm.ExternalDeliverPrefix)
		}
	}

	if a.Name != b.Name {
		diff("Name", a.Name, b.Name)
	}
	if a.Description != b.Description {
		diff("Description", a.Description, b.Description)
	}
	if !reflect.DeepEqual(a.Subjects, b.Subjects) {
		diff("Subjects", a.Subjects, b.Subjects)
	}
	if strings.ToLower(a.Retention.String()) != strings.ToLower(b.Retention) {
		diff("Retention", a.Retention, b.Retention)
	}
	if a.MaxConsumers != b.MaxConsumers {
		diff("MaxConsumers", a.MaxConsumers, b.MaxConsumers)
	}
	if a.MaxMsgs != int64(b.MaxMsgs) {
		diff("MaxMsgs", a.MaxMsgs, b.MaxMsgs)
	}
	if a.MaxBytes != int64(b.MaxBytes) {
		diff("MaxBytes", a.MaxBytes, b.MaxBytes)
	}
	if strings.ToLower(a.Discard.String()) != strings.ToLower("discard"+b.Discard) {
		diff("Discard", a.Discard, b.Discard)
	}
	if b.MaxAge == "" {
		b.MaxAge = "0s"
	}
	if d, err := time.ParseDuration(b.MaxAge); err == nil && a.MaxAge != d {
		diff("MaxAge", a.MaxAge, b.MaxAge)
	} else if err != nil {
		diff("MaxAge", a.MaxAge, err)
	}
	if a.MaxMsgsPerSubject != int64(b.MaxMsgsPerSubject) {
		diff("MaxMsgsPerSubject", a.MaxMsgsPerSubject, b.MaxMsgsPerSubject)
	}
	if a.MaxMsgSize != int32(b.MaxMsgSize) {
		diff("MaxMsgSize", a.MaxMsgSize, b.MaxMsgSize)
	}
	if strings.ToLower(a.Storage.String()) != strings.ToLower(b.Storage) {
		diff("Storage", a.Storage, b.Storage)
	}
	if a.Replicas != b.Replicas {
		diff("Replicas", a.Replicas, b.Replicas)
	}
	if a.NoAck != b.NoAck {
		diff("NoAck", a.NoAck, b.NoAck)
	}

	if b.DuplicateWindow == "" && a.Mirror != nil {
		b.DuplicateWindow = "2m0s"
	} else if b.DuplicateWindow == "" && a.Mirror == nil {
		b.DuplicateWindow = "0s"
	}
	if d, err := time.ParseDuration(b.DuplicateWindow); err == nil && a.Duplicates != d {
		diff("Duplicates", a.Duplicates, b.DuplicateWindow)
	} else if err != nil {
		diff("Duplicates", a.Duplicates, err)
	}

	if got, want := a.Placement != nil, b.Placement != nil; got != want {
		diff("Placement", got, want)
	} else if got && want {
		ap, bp := a.Placement, b.Placement
		if ap.Cluster != bp.Cluster {
			diff("Placement.Cluster", ap.Cluster, bp.Cluster)
		}
		if !reflect.DeepEqual(ap.Tags, bp.Tags) {
			diff("Placement.Tags", ap.Tags, bp.Tags)
		}
	}

	if got, want := a.Mirror != nil, b.Mirror != nil; got != want {
		diff("Mirror", got, want)
	} else if got && want {
		am, bm := a.Mirror, b.Mirror
		checkSource(am, bm)
	}

	if len(a.Sources) != len(b.Sources) {
		diff("Sources", len(a.Sources), len(b.Sources))
	} else {
		for i := 0; i < len(a.Sources); i++ {
			as, bs := a.Sources[i], b.Sources[i]
			checkSource(as, bs)
		}
	}

	if len(msgs) == 0 {
		return nil
	}

	return fmt.Errorf("config and spec not equal: %s", strings.Join(msgs, ", "))
}

func getStreamSpec(w io.Writer, streamName string) (v1beta2.StreamSpec, error) {
	rc, err := runToolStdoutPipe(
		w,
		"kubectl",
		"get",
		"--output",
		"json",
		"stream",
		streamName,
	)
	if err != nil {
		return v1beta2.StreamSpec{}, err
	}
	defer rc.Close()

	var data v1beta2.Stream
	if err := json.NewDecoder(rc).Decode(&data); err != nil {
		return v1beta2.StreamSpec{}, fmt.Errorf("failed to decode stream spec: %w", err)
	}
	return data.Spec, nil
}

func getStreamConfig(w io.Writer, streamName string) (nats.StreamConfig, error) {
	rc, err := runToolStdoutPipe(
		w,
		"kubectl",
		"exec",
		"deployment/nats-box",
		"--",
		"nats",
		"stream",
		"info",
		"--json",
		streamName,
	)
	if err != nil {
		return nats.StreamConfig{}, err
	}
	defer rc.Close()

	var data nats.StreamInfo
	if err := json.NewDecoder(rc).Decode(&data); err != nil {
		return nats.StreamConfig{}, fmt.Errorf("failed to decode stream config: %w", err)
	}
	return data.Config, nil
}

func kubectlApply(w io.Writer, file string) error {
	return runTool(w, "kubectl", "apply", "--filename", file)
}

func kubectlDelete(w io.Writer, file string) error {
	return runTool(w, "kubectl", "delete", "--filename", file)
}

func installStream(w io.Writer, name, specFile string) (err error) {
	cmds := [][]string{
		{"kubectl", "apply", "--filename", specFile},
		{"kubectl", "wait", "--for=condition=Ready", "--timeout=10s", "stream", name},
	}

	for _, c := range cmds {
		err := retry(10, 3*time.Second, func() error {
			return runTool(w, c[0], c[1:]...)
		})
		if err != nil {
			uninstallStream(w, specFile)
			return fmt.Errorf("failed to install Stream: %w", err)
		}
	}

	return nil
}

func uninstallStream(w io.Writer, specFile string) error {
	cmds := [][]string{
		{"kubectl", "delete", "--filename", specFile},
	}

	for _, c := range cmds {
		err := retry(10, 3*time.Second, func() error {
			return runTool(w, c[0], c[1:]...)
		})
		if err != nil {
			return fmt.Errorf("failed to uninstall Stream: %w", err)
		}
	}

	return nil
}

func installNACK(w io.Writer, helmValues string) (err error) {
	cmds := [][]string{
		{"helm", "install", "--values", helmValues, "nack", "nats/nack"},
		{"kubectl", "wait", "--for=condition=Available", "--all", "--timeout=120s", "deployment", "nack"},
	}

	for _, c := range cmds {
		err := retry(10, 3*time.Second, func() error {
			return runTool(w, c[0], c[1:]...)
		})
		if err != nil {
			uninstallNACK(w)
			return fmt.Errorf("failed to install nack: %w", err)
		}
	}

	return nil
}

func uninstallNACK(w io.Writer) error {
	cmds := [][]string{
		{"helm", "uninstall", "nack"},
		{"kubectl", "wait", "--for=delete", "--all", "--timeout=500s", "deployment", "nack"},
	}

	for _, c := range cmds {
		err := retry(10, 3*time.Second, func() error {
			return runTool(w, c[0], c[1:]...)
		})
		if err != nil {
			return fmt.Errorf("failed to uninstall nack: %w", err)
		}
	}

	return nil
}

func installNATS(w io.Writer, helmValues string) error {
	cmds := [][]string{
		{"helm", "install", "--values", helmValues, "nats", "nats/nats"},
		// This can fail with pod-not-found, needs retries.
		{"kubectl", "wait", "--for=condition=Ready", "--all", "--timeout=120s", "pod", "nats-0"},
	}

	for _, c := range cmds {
		err := retry(10, 3*time.Second, func() error {
			return runTool(w, c[0], c[1:]...)
		})
		if err != nil {
			uninstallNATS(w)
			return fmt.Errorf("failed to install NATS Server: %w", err)
		}
	}

	return nil
}

func uninstallNATS(w io.Writer) error {
	cmds := [][]string{
		{"helm", "uninstall", "nats"},
		{"kubectl", "wait", "--for=delete", "--all", "--timeout=500s", "pod", "nats-0"},
	}

	for _, c := range cmds {
		err := retry(10, 3*time.Second, func() error {
			return runTool(w, c[0], c[1:]...)
		})
		if err != nil {
			return fmt.Errorf("failed to uninstall NATS Server: %w", err)
		}
	}

	return nil
}

func errCheck(t *testing.T, fn func() error) {
	t.Helper()
	if err := fn(); err != nil {
		t.Fatal(err)
	}
}

func helmRepoAdd(w io.Writer, name, repoURL string) error {
	err := runTool(w,
		"helm", "repo", "add", "--force-update", name, repoURL)
	if err != nil {
		return fmt.Errorf("failed to install %q repo: %w", name, err)
	}

	return nil
}

func helmRepoRemove(w io.Writer, name string) error {
	err := runTool(w,
		"helm", "repo", "remove", name)
	if err != nil {
		return fmt.Errorf("failed to uninstall %q repo: %w", name, err)
	}

	return nil
}

func runToolStdoutPipe(w io.Writer, name string, args ...string) (io.ReadCloser, error) {
	cmd, err := toolCmd(name, args...)
	if err != nil {
		return nil, err
	}

	if lw, ok := w.(*logfWriter); ok {
		w = &logfWriter{logger: lw.logger, prefix: cmd.String()}
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

	if lw, ok := w.(*logfWriter); ok {
		w = &logfWriter{logger: lw.logger, prefix: cmd.String()}
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
		fmt.Sprintf("HOME=%s", toolDir),
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
