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

	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta1"
	clientset "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned"
	scheme "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/scheme"
	typed "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/typed/jetstream/v1beta1"
	informers "github.com/nats-io/nack/pkg/jetstream/generated/informers/externalversions"
	listers "github.com/nats-io/nack/pkg/jetstream/generated/listers/jetstream/v1beta1"

	k8sapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
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

	// readyCondType is the Ready condition type.
	readyCondType = "Ready"
)

type Options struct {
	Ctx context.Context

	KubeIface      kubernetes.Interface
	JetstreamIface clientset.Interface

	NATSClientName  string
	NATSCredentials string
	NATSServerURL   string

	Recorder record.EventRecorder
}

type Controller struct {
	ctx context.Context

	ki              k8styped.CoreV1Interface
	ji              typed.JetstreamV1beta1Interface
	informerFactory informers.SharedInformerFactory
	rec             record.EventRecorder

	natsName string

	strLister listers.StreamLister
	strSynced cache.InformerSynced
	strQueue  workqueue.RateLimitingInterface

	strTmplLister listers.StreamTemplateLister
	strTmplSynced cache.InformerSynced
	strTmplQueue  workqueue.RateLimitingInterface

	cnsLister listers.ConsumerLister
	cnsSynced cache.InformerSynced
	cnsQueue  workqueue.RateLimitingInterface
}

func NewController(opt Options) *Controller {
	resyncPeriod := 30 * time.Second
	informerFactory := informers.NewSharedInformerFactory(opt.JetstreamIface, resyncPeriod)

	streamInformer := informerFactory.Jetstream().V1beta1().Streams()
	streamTmplInformer := informerFactory.Jetstream().V1beta1().StreamTemplates()
	consumerInformer := informerFactory.Jetstream().V1beta1().Consumers()

	if opt.Recorder == nil {
		utilruntime.Must(scheme.AddToScheme(k8sscheme.Scheme))
		eventBroadcaster := record.NewBroadcaster()
		eventBroadcaster.StartLogging(klog.Infof)
		eventBroadcaster.StartRecordingToSink(&k8styped.EventSinkImpl{
			Interface: opt.KubeIface.CoreV1().Events(""),
		})

		opt.Recorder = eventBroadcaster.NewRecorder(k8sscheme.Scheme, k8sapi.EventSource{
			Component: "jetstream-controller",
		})
	}

	if opt.NATSClientName == "" {
		opt.NATSClientName = "jetstream-controller"
	}

	ji := opt.JetstreamIface.JetstreamV1beta1()
	streamQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Streams")
	streamTmplQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Streams")
	consumerQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Consumers")

	streamInformer.Informer().AddEventHandler(eventHandlers(
		opt.Ctx,
		streamQueue,
	))

	streamTmplInformer.Informer().AddEventHandler(eventHandlers(
		opt.Ctx,
		streamTmplQueue,
	))

	consumerInformer.Informer().AddEventHandler(eventHandlers(
		opt.Ctx,
		consumerQueue,
	))

	return &Controller{
		ctx: opt.Ctx,

		ki:              opt.KubeIface.CoreV1(),
		ji:              ji,
		informerFactory: informerFactory,
		rec:             opt.Recorder,

		natsName: opt.NATSClientName,

		strLister: streamInformer.Lister(),
		strSynced: streamInformer.Informer().HasSynced,
		strQueue:  streamQueue,

		strTmplLister: streamTmplInformer.Lister(),
		strTmplSynced: streamTmplInformer.Informer().HasSynced,
		strTmplQueue:  streamTmplQueue,

		cnsLister: consumerInformer.Lister(),
		cnsSynced: consumerInformer.Informer().HasSynced,
		cnsQueue:  consumerQueue,
	}
}

func (c *Controller) Run() error {
	defer utilruntime.HandleCrash()

	defer c.strQueue.ShutDown()
	defer c.strTmplQueue.ShutDown()
	defer c.cnsQueue.ShutDown()

	c.informerFactory.Start(c.ctx.Done())

	if !cache.WaitForCacheSync(c.ctx.Done(), c.strSynced) {
		return fmt.Errorf("failed to wait for stream cache sync")
	}
	if !cache.WaitForCacheSync(c.ctx.Done(), c.strTmplSynced) {
		return fmt.Errorf("failed to wait for stream template cache sync")
	}
	if !cache.WaitForCacheSync(c.ctx.Done(), c.cnsSynced) {
		return fmt.Errorf("failed to wait for consumer cache sync")
	}

	go wait.Until(c.runStreamQueue, time.Second, c.ctx.Done())
	go wait.Until(c.runStreamTemplateQueue, time.Second, c.ctx.Done())
	go wait.Until(c.runConsumerQueue, time.Second, c.ctx.Done())

	<-c.ctx.Done()
	// Gracefully shutdown.
	return nil
}

func (c *Controller) normalEvent(o runtime.Object, reason, message string) {
	if c.rec != nil {
		c.rec.Event(o, k8sapi.EventTypeNormal, reason, message)
	}
}

func (c *Controller) warningEvent(o runtime.Object, reason, message string) {
	if c.rec != nil {
		c.rec.Event(o, k8sapi.EventTypeWarning, reason, message)
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

func getCreds(ctx context.Context, sec apis.CredentialsSecret, sif k8styped.SecretInterface) ([]byte, error) {
	if sec.Name == "" || sec.Key == "" {
		return nil, nil
	}

	return getSecret(ctx, sec.Name, sec.Key, sif)
}

func getSecret(ctx context.Context, name, key string, sif k8styped.SecretInterface) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sec, err := sif.Get(ctx, name, k8smeta.GetOptions{})
	if err != nil {
		return nil, err
	}

	return sec.Data[key], nil
}

func enqueueWork(q workqueue.RateLimitingInterface, item interface{}) (err error) {
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		return fmt.Errorf("failed to enqueue work: %w", err)
	}

	q.Add(key)
	return nil
}

type processorFunc func(ns, name string, c jsmClient) error

func processQueueNext(q workqueue.RateLimitingInterface, c jsmClient, process processorFunc) {
	item, shutdown := q.Get()
	if shutdown {
		return
	}
	defer q.Done(item)

	ns, name, err := splitNamespaceName(item)
	if err != nil {
		// Probably junk, clean it up.
		utilruntime.HandleError(err)
		q.Forget(item)
		return
	}

	err = process(ns, name, c)
	if err == nil {
		// Item processed successfully, don't requeue.
		q.Forget(item)
		return
	}

	utilruntime.HandleError(err)

	if q.NumRequeues(item) < maxQueueRetries {
		// Failed to process item, try again.
		q.AddRateLimited(item)
		return
	}

	// If we haven't been able to recover by this point, then just stop.
	// The user should have enough info in kubectl describe to debug.
	q.Forget(item)
}

func upsertCondition(cs []apis.Condition, next apis.Condition) []apis.Condition {
	for i := 0; i < len(cs); i++ {
		if cs[i].Type != next.Type {
			continue
		}

		cs[i] = next
		return cs
	}

	return append(cs, next)
}

func shouldEnqueue(prevObj, nextObj interface{}) bool {
	type crd interface {
		GetDeletionTimestamp() *k8smeta.Time
		GetSpec() interface{}
	}

	prev, ok := prevObj.(crd)
	if !ok {
		return false
	}

	next, ok := nextObj.(crd)
	if !ok {
		return false
	}

	markedDelete := next.GetDeletionTimestamp() != nil
	specChanged := !equality.Semantic.DeepEqual(prev.GetSpec(), next.GetSpec())

	return markedDelete || specChanged
}

func eventHandlers(ctx context.Context, q workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := enqueueWork(q, obj); err != nil {
				utilruntime.HandleError(err)
			}
		},
		UpdateFunc: func(prev, next interface{}) {
			if !shouldEnqueue(prev, next) {
				return
			}

			if err := enqueueWork(q, next); err != nil {
				utilruntime.HandleError(err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if err := enqueueWork(q, obj); err != nil {
				utilruntime.HandleError(err)
			}
		},
	}
}
