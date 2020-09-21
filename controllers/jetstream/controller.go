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

package jetstream

import (
	"context"
	"fmt"
	"strings"
	"time"

	jsmapi "github.com/nats-io/jsm.go/api"
	"github.com/nats-io/jwt"
	"github.com/nats-io/nats.go"

	clientset "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned"
	scheme "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/scheme"
	typed "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/typed/jetstream/v1"
	informers "github.com/nats-io/nack/pkg/jetstream/generated/informers/externalversions"
	listers "github.com/nats-io/nack/pkg/jetstream/generated/listers/jetstream/v1"

	k8sapi "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	corev1types "k8s.io/client-go/kubernetes/typed/core/v1"
	k8styped "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	klog "k8s.io/klog/v2"
)

const (
	// maxQueueRetries is the max times an item will be retried. An item will
	// be pulled maxQueueRetries+1 times from the queue. On pull number
	// maxQueueRetries+1, if it fails again, it won't be retried.
	maxQueueRetries = 10
)

type Options struct {
	Ctx context.Context

	KubeIface      kubernetes.Interface
	JetstreamIface clientset.Interface

	NATSClientName string
}

type Controller struct {
	ctx context.Context

	ki k8styped.CoreV1Interface
	ji typed.JetstreamV1Interface

	informerFactory informers.SharedInformerFactory
	streamLister    listers.StreamLister
	streamSynced    cache.InformerSynced
	streamQueue     workqueue.RateLimitingInterface

	rec record.EventRecorder

	natsName string
	sc       streamClient
}

func NewController(opt Options) (*Controller, error) {
	resyncPeriod := 30 * time.Second
	informerFactory := informers.NewSharedInformerFactory(opt.JetstreamIface, resyncPeriod)
	streamInformer := informerFactory.Jetstream().V1().Streams()

	utilruntime.Must(scheme.AddToScheme(k8sscheme.Scheme))
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&k8styped.EventSinkImpl{
		Interface: opt.KubeIface.CoreV1().Events(""),
	})

	ctrl := &Controller{
		ctx:             opt.Ctx,
		ki:              opt.KubeIface.CoreV1(),
		ji:              opt.JetstreamIface.JetstreamV1(),
		informerFactory: informerFactory,

		streamLister: streamInformer.Lister(),
		streamSynced: streamInformer.Informer().HasSynced,
		streamQueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Streams"),

		rec: eventBroadcaster.NewRecorder(k8sscheme.Scheme, k8sapi.EventSource{
			Component: "jetstream-controller",
		}),

		natsName: opt.NATSClientName,
		sc:       &realStreamClient{},
	}
	if ctrl.natsName == "" {
		ctrl.natsName = "jetstream-controller"
	}

	streamInformer.Informer().AddEventHandler(streamEventHandlers(
		ctrl.ctx,
		ctrl.streamQueue,
		ctrl.ji,
	))

	return ctrl, nil
}

func (c *Controller) Run() error {
	defer utilruntime.HandleCrash()
	defer c.streamQueue.ShutDown()

	c.informerFactory.Start(c.ctx.Done())

	if !cache.WaitForCacheSync(c.ctx.Done(), c.streamSynced) {
		return fmt.Errorf("failed to wait for cache sync")
	}

	go wait.Until(c.runStreamQueue, time.Second, c.ctx.Done())

	<-c.ctx.Done()
	// Gracefully shutdown.
	return nil
}

func (c *Controller) normalEvent(o runtime.Object, reason, message string) {
	if c.rec != nil {
		c.rec.Event(o, k8sapi.EventTypeNormal, reason, message)
	}
}

func splitNamespaceName(item interface{}) (ns string, name string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to split namespace-name: %w", err)
		}
	}()

	key, ok := item.(string)
	if !ok {
		return "", "", fmt.Errorf("unexpected type: got=%T, want=%T", item, key)
	}

	ns, name, err = cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return "", "", err
	}

	return ns, name, nil
}

func getStorageType(s string) (jsmapi.StorageType, error) {
	switch s {
	case strings.ToLower(jsmapi.FileStorage.String()):
		return jsmapi.FileStorage, nil
	case strings.ToLower(jsmapi.MemoryStorage.String()):
		return jsmapi.MemoryStorage, nil
	default:
		return 0, fmt.Errorf("invalid jetstream storage option: %s", s)
	}
}

func wipeSlice(buf []byte) {
	for i := range buf {
		buf[i] = 'x'
	}
}

func getNATSOptions(connName string, creds []byte) []nats.Option {
	userCB := func() (string, error) {
		return jwt.ParseDecoratedJWT(creds)
	}
	sigCB := func(nonce []byte) ([]byte, error) {
		defer wipeSlice(creds)
		keys, err := jwt.ParseDecoratedNKey(creds)
		if err != nil {
			return nil, err
		}
		defer keys.Wipe()

		sig, _ := keys.Sign(nonce)
		return sig, nil
	}

	opts := []nats.Option{
		nats.Name(connName),
		nats.Option(func(o *nats.Options) error {
			o.Pedantic = true
			return nil

		}),
	}

	if creds != nil {
		opts = append(opts, nats.UserJWT(userCB, sigCB))
	}

	return opts
}

func addFinalizer(fs []string, key string) []string {
	for _, f := range fs {
		if f == key {
			return fs
		}
	}

	return append(fs, key)
}

func removeFinalizer(fs []string, key string) []string {
	var filtered []string
	for _, f := range fs {
		if f == key {
			continue
		}
		filtered = append(filtered, f)
	}

	return filtered
}

func getSecret(ctx context.Context, name, key string, sif corev1types.SecretInterface) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sec, err := sif.Get(ctx, name, k8smeta.GetOptions{})
	if err != nil {
		return nil, err
	}

	return sec.Data[key], nil
}
