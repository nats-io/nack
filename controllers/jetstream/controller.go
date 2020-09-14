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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"
	"github.com/sirupsen/logrus"

	scheme "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/scheme"
	typed "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/typed/jetstream/v1"
	extinformers "github.com/nats-io/nack/pkg/jetstream/generated/informers/externalversions"
	listers "github.com/nats-io/nack/pkg/jetstream/generated/listers/jetstream/v1"

	k8sapi "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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
)

var (
	errNothingToUpdate = errors.New("nothing to update")
)

type Options struct {
	Ctx context.Context

	KubeIface      kubernetes.Interface
	APIExtIface    apiextensionsclientset.Interface
	JetstreamIface typed.JetstreamV1Interface

	InformerFactory extinformers.SharedInformerFactory

	NATSClientName string
}

type Controller struct {
	ctx context.Context

	// Clients
	infoFactory extinformers.SharedInformerFactory
	ji          typed.JetstreamV1Interface

	streamLister listers.StreamLister
	streamSynced cache.InformerSynced
	streamQueue  workqueue.RateLimitingInterface
	streamMap    map[string]*jsm.Stream
	rec          record.EventRecorder

	natsName string
}

func NewController(opt Options) (*Controller, error) {
	streamInformer := opt.InformerFactory.Jetstream().V1().Streams()

	utilruntime.Must(scheme.AddToScheme(k8sscheme.Scheme))
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logrus.Infof)
	eventBroadcaster.StartRecordingToSink(&k8styped.EventSinkImpl{
		Interface: opt.KubeIface.CoreV1().Events(""),
	})

	if err := createStreamCRD(opt.APIExtIface); err != nil {
		return nil, err
	}

	ctrl := &Controller{
		ctx:         opt.Ctx,
		natsName:    opt.NATSClientName,
		ji:          opt.JetstreamIface,
		infoFactory: opt.InformerFactory,

		streamLister: streamInformer.Lister(),
		streamSynced: streamInformer.Informer().HasSynced,
		streamQueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Streams"),
		streamMap:    make(map[string]*jsm.Stream),

		rec: eventBroadcaster.NewRecorder(k8sscheme.Scheme, k8sapi.EventSource{
			Component: "jetstream-controller",
		}),
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

	c.infoFactory.Start(c.ctx.Done())

	if !cache.WaitForCacheSync(c.ctx.Done(), c.streamSynced) {
		return fmt.Errorf("failed to wait for cache sync")
	}

	go wait.Until(c.streamWorker, time.Second, c.ctx.Done())

	<-c.ctx.Done()
	// Gracefully shutdown.
	return nil
}

func (c *Controller) normalEvent(o runtime.Object, reason, message string) {
	c.rec.Event(o, k8sapi.EventTypeNormal, reason, message)
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

func hasFinalizerKey(finalizers []string, key string) bool {
	for _, f := range finalizers {
		if f == key {
			return true
		}
	}

	return false
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
