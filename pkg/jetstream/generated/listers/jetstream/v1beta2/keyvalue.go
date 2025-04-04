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

// KeyValueLister helps list KeyValues.
// All objects returned here must be treated as read-only.
type KeyValueLister interface {
	// List lists all KeyValues in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*jetstreamv1beta2.KeyValue, err error)
	// KeyValues returns an object that can list and get KeyValues.
	KeyValues(namespace string) KeyValueNamespaceLister
	KeyValueListerExpansion
}

// keyValueLister implements the KeyValueLister interface.
type keyValueLister struct {
	listers.ResourceIndexer[*jetstreamv1beta2.KeyValue]
}

// NewKeyValueLister returns a new KeyValueLister.
func NewKeyValueLister(indexer cache.Indexer) KeyValueLister {
	return &keyValueLister{listers.New[*jetstreamv1beta2.KeyValue](indexer, jetstreamv1beta2.Resource("keyvalue"))}
}

// KeyValues returns an object that can list and get KeyValues.
func (s *keyValueLister) KeyValues(namespace string) KeyValueNamespaceLister {
	return keyValueNamespaceLister{listers.NewNamespaced[*jetstreamv1beta2.KeyValue](s.ResourceIndexer, namespace)}
}

// KeyValueNamespaceLister helps list and get KeyValues.
// All objects returned here must be treated as read-only.
type KeyValueNamespaceLister interface {
	// List lists all KeyValues in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*jetstreamv1beta2.KeyValue, err error)
	// Get retrieves the KeyValue from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*jetstreamv1beta2.KeyValue, error)
	KeyValueNamespaceListerExpansion
}

// keyValueNamespaceLister implements the KeyValueNamespaceLister
// interface.
type keyValueNamespaceLister struct {
	listers.ResourceIndexer[*jetstreamv1beta2.KeyValue]
}
