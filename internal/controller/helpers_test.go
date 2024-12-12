package controller

import (
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"os"
	"time"
)

func assertReadyStateMatches(condition api.Condition, status v1.ConditionStatus, reason string, message string, transitionTime time.Time) {
	GinkgoHelper()

	Expect(condition.Type).To(Equal(readyCondType))
	Expect(condition.Status).To(Equal(status))
	Expect(condition.Reason).To(Equal(reason))
	Expect(condition.Message).To(ContainSubstring(message))

	// Assert valid transition time
	t, err := time.Parse(time.RFC3339Nano, condition.LastTransitionTime)
	Expect(err).NotTo(HaveOccurred())
	Expect(t).To(BeTemporally("~", transitionTime, time.Second))
}

func CreateTestServer() *server.Server {
	opts := &natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.Debug = true

	dir, err := os.MkdirTemp("", "nats-*")
	Expect(err).NotTo(HaveOccurred())
	opts.StoreDir = dir

	ns := natsserver.RunServer(opts)
	Expect(err).NotTo(HaveOccurred())

	return ns
}
