// Copyright 2025 The NATS Authors
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

// Code generated by informer-gen. DO NOT EDIT.

package v1beta2

import (
	context "context"
	time "time"

	apisjetstreamv1beta2 "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	versioned "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned"
	internalinterfaces "github.com/nats-io/nack/pkg/jetstream/generated/informers/externalversions/internalinterfaces"
	jetstreamv1beta2 "github.com/nats-io/nack/pkg/jetstream/generated/listers/jetstream/v1beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ObjectStoreInformer provides access to a shared informer and lister for
// ObjectStores.
type ObjectStoreInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() jetstreamv1beta2.ObjectStoreLister
}

type objectStoreInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewObjectStoreInformer constructs a new informer for ObjectStore type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewObjectStoreInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredObjectStoreInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredObjectStoreInformer constructs a new informer for ObjectStore type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredObjectStoreInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.JetstreamV1beta2().ObjectStores(namespace).List(context.Background(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.JetstreamV1beta2().ObjectStores(namespace).Watch(context.Background(), options)
			},
			ListWithContextFunc: func(ctx context.Context, options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.JetstreamV1beta2().ObjectStores(namespace).List(ctx, options)
			},
			WatchFuncWithContext: func(ctx context.Context, options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.JetstreamV1beta2().ObjectStores(namespace).Watch(ctx, options)
			},
		},
		&apisjetstreamv1beta2.ObjectStore{},
		resyncPeriod,
		indexers,
	)
}

func (f *objectStoreInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredObjectStoreInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *objectStoreInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apisjetstreamv1beta2.ObjectStore{}, f.defaultInformer)
}

func (f *objectStoreInformer) Lister() jetstreamv1beta2.ObjectStoreLister {
	return jetstreamv1beta2.NewObjectStoreLister(f.Informer().GetIndexer())
}
