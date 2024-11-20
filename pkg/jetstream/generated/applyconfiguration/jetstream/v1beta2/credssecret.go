// Copyright 2024 The NATS Authors
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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1beta2

// CredsSecretApplyConfiguration represents a declarative configuration of the CredsSecret type for use
// with apply.
type CredsSecretApplyConfiguration struct {
	File   *string                      `json:"file,omitempty"`
	Secret *SecretRefApplyConfiguration `json:"secret,omitempty"`
}

// CredsSecretApplyConfiguration constructs a declarative configuration of the CredsSecret type for use with
// apply.
func CredsSecret() *CredsSecretApplyConfiguration {
	return &CredsSecretApplyConfiguration{}
}

// WithFile sets the File field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the File field is set to the value of the last call.
func (b *CredsSecretApplyConfiguration) WithFile(value string) *CredsSecretApplyConfiguration {
	b.File = &value
	return b
}

// WithSecret sets the Secret field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Secret field is set to the value of the last call.
func (b *CredsSecretApplyConfiguration) WithSecret(value *SecretRefApplyConfiguration) *CredsSecretApplyConfiguration {
	b.Secret = value
	return b
}
