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

// Code generated by lister-gen. DO NOT EDIT.

package v1beta2

import (
	jetstreamv1beta2 "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// ConsumerLister helps list Consumers.
// All objects returned here must be treated as read-only.
type ConsumerLister interface {
	// List lists all Consumers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*jetstreamv1beta2.Consumer, err error)
	// Consumers returns an object that can list and get Consumers.
	Consumers(namespace string) ConsumerNamespaceLister
	ConsumerListerExpansion
}

// consumerLister implements the ConsumerLister interface.
type consumerLister struct {
	listers.ResourceIndexer[*jetstreamv1beta2.Consumer]
}

// NewConsumerLister returns a new ConsumerLister.
func NewConsumerLister(indexer cache.Indexer) ConsumerLister {
	return &consumerLister{listers.New[*jetstreamv1beta2.Consumer](indexer, jetstreamv1beta2.Resource("consumer"))}
}

// Consumers returns an object that can list and get Consumers.
func (s *consumerLister) Consumers(namespace string) ConsumerNamespaceLister {
	return consumerNamespaceLister{listers.NewNamespaced[*jetstreamv1beta2.Consumer](s.ResourceIndexer, namespace)}
}

// ConsumerNamespaceLister helps list and get Consumers.
// All objects returned here must be treated as read-only.
type ConsumerNamespaceLister interface {
	// List lists all Consumers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*jetstreamv1beta2.Consumer, err error)
	// Get retrieves the Consumer from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*jetstreamv1beta2.Consumer, error)
	ConsumerNamespaceListerExpansion
}

// consumerNamespaceLister implements the ConsumerNamespaceLister
// interface.
type consumerNamespaceLister struct {
	listers.ResourceIndexer[*jetstreamv1beta2.Consumer]
}
