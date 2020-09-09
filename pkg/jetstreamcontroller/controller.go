// Copyright 2020 The NATS Authors
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

package jetstreamcontroller

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	jsapiv1 "github.com/nats-io/nack/pkg/jetstreamcontroller/apis/jetstreamcontroller/v1alpha1"
	jsk8sclient "github.com/nats-io/nack/pkg/jetstreamcontroller/generated/clientset/versioned"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfields "k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	// Load all auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const Version = "0.1.0"

const (
	DefaultQueueGroupName = "controllers"
)

// Options are the options for the controller.
type Options struct {
	// ClusterName is the NATS cluster name.
	ClusterName string

	// NoSignals marks whether to enable the signal handler.
	NoSignals bool

	// NatsServerURL is the address of the ground NATS Server.
	NatsServerURL string

	// NatsCredentials are the auth keys for the NATS Server.
	NatsCredentials string

	// PodNamespace is the namespace where this controller is running.
	PodNamespace string
}

// Controller manages NATS Leaf node clusters running in Kubernetes.
type Controller struct {
	mu sync.Mutex

	// client to interact with Kubernetes resources.
	kc kubernetes.Interface

	// client to interact with JetStream CRDs.
	jsc jsk8sclient.Interface

	// opts is the set of options.
	opts *Options

	// nc is the NATS connection.
	nc *nats.Conn

	// quit stops the controller.
	quit func()

	// shutdown is to shutdown the controller.
	shutdown bool
}

// NewController creats a new Controller.
func NewController(opts *Options) *Controller {
	if opts == nil {
		opts = &Options{}
	}
	ns := os.Getenv("POD_NAMESPACE")
	if ns != "" {
		opts.PodNamespace = ns
	} else {
		opts.PodNamespace = "default"
	}
	return &Controller{
		opts: opts,
	}
}

// Run starts the controller.
func (c *Controller) Run(ctx context.Context) error {
	err := c.setupK8S()
	if err != nil {
		return err
	}

	err = c.setupNATS()
	if err != nil {
		return err
	}

	// Initial refresh, find all Stream CRD that have been created on start.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	streams, err := c.jsc.JetstreamcontrollerV1alpha1().Streams("default").List(ctx, v1.ListOptions{})
	if err != nil {
		log.Errorf("Error: %s", err)
	} else {
		// Make sure that the Stream CRDs are mapped to a Jetstream Stream.
		for _, stream := range streams.Items {
			name := stream.Spec.StreamName
			log.Infof("Found stream: %s", name)
			streamInfo, err := c.nc.RequestWithContext(ctx, "$JS.API.STREAM.INFO.ORDERS", []byte(""))
			if err != nil {
				// If not found error, then start task to create the stream as per the Stream CRD definition.
				log.Infof("Stream named %q was not found, creating it...", stream.Spec.StreamName)
			} else {
				log.Debugf("Found Stream: %+v", string(streamInfo.Data))
			}
		}
	}

	// Run until context is cancelled via a signal.x
	ctx, cancel = context.WithCancel(ctx)
	c.quit = func() {
		// Signal cancellation of the main context.
		cancel()
	}
	if !c.opts.NoSignals {
		go c.SetupSignalHandler(ctx)
	}

	//
	// Start watches for streams and consumers.
	//
	_, informer := NewInformer(c, k8scache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			log.Infof("New stream? Maybe: %v", o)
			// TODO
		},
		UpdateFunc: func(o interface{}, n interface{}) {
			// TODO
		},
		DeleteFunc: func(o interface{}) {
			// TODO
		},
	}, 5*time.Second)

	// Wait for context to get cancelled or get a signal.
	informer.Run(ctx.Done())

	return nil
}

func (c *Controller) setupNATS() error {
	// Create subscriptions that can be used to make updates,
	// to the configuration map.
	nopts := make([]nats.Option, 0)
	nopts = append(nopts, nats.Name(fmt.Sprintf("jetstream-controller:%s", c.opts.ClusterName)))
	nc, err := nats.Connect(c.opts.NatsServerURL, nopts...)
	if err != nil {
		return err
	}

	c.nc = nc
	return nil
}

func (c *Controller) setupK8S() error {
	// Creates controller cluster config.
	var err error
	var config *rest.Config
	if kubeconfig := os.Getenv("KUBERNETES_CONFIG_FILE"); kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return err
	}

	// Client for the K8S API.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	c.kc = clientset

	jsc, err := jsk8sclient.NewForConfig(config)
	if err != nil {
		return err
	}
	c.jsc = jsc

	return nil
}

// NewInformer takes a controller and a set of resource handlers and
// returns an indexer and a controller that are subscribed to changes
// to the state of a NATS cluster resource.
func NewInformer(
	c *Controller,
	resourceFuncs k8scache.ResourceEventHandlerFuncs,
	interval time.Duration,
) (k8scache.Indexer, k8scache.Controller) {
	listWatcher := k8scache.NewListWatchFromClient(
		c.jsc.JetstreamcontrollerV1alpha1().RESTClient(),

		// Plural name of the CRD
		"streams",

		// Namespace where the clusters will be created.
		c.opts.PodNamespace,
		k8sfields.Everything(),
	)
	return k8scache.NewIndexerInformer(
		listWatcher,
		&jsapiv1.Stream{},

		// How often it will poll for the state
		// of the resources.
		interval,

		// Handlers
		resourceFuncs,
		k8scache.Indexers{},
	)
}

// SetupSignalHandler enables handling process signals.
func (c *Controller) SetupSignalHandler(ctx context.Context) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for sig := range sigCh {
		log.Debugf("Trapped '%v' signal", sig)

		// If main context already done, then just skip
		select {
		case <-ctx.Done():
			continue
		default:
		}

		switch sig {
		case syscall.SIGINT:
			log.Infof("Exiting...")
			os.Exit(0)
			return
		case syscall.SIGTERM:
			// Gracefully shutdown the operator.  This blocks
			// until all controllers have stopped running.
			c.Shutdown()
			return
		}
	}
}

// Shutdown stops the operator controller.
func (c *Controller) Shutdown() {
	c.mu.Lock()
	if c.shutdown {
		c.mu.Unlock()
		return
	}
	c.shutdown = true
	nc := c.nc
	c.mu.Unlock()

	err := nc.Drain()
	log.Errorf("Error disconnecting from NATS: %s", err)

	c.quit()
	log.Infof("Bye...")
	return
}
