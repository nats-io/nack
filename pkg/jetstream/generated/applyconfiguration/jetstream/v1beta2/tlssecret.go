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

// TLSSecretApplyConfiguration represents a declarative configuration of the TLSSecret type for use
// with apply.
type TLSSecretApplyConfiguration struct {
	ClientCert *string                      `json:"cert,omitempty"`
	ClientKey  *string                      `json:"key,omitempty"`
	RootCAs    *string                      `json:"ca,omitempty"`
	Secret     *SecretRefApplyConfiguration `json:"secret,omitempty"`
}

// TLSSecretApplyConfiguration constructs a declarative configuration of the TLSSecret type for use with
// apply.
func TLSSecret() *TLSSecretApplyConfiguration {
	return &TLSSecretApplyConfiguration{}
}

// WithClientCert sets the ClientCert field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClientCert field is set to the value of the last call.
func (b *TLSSecretApplyConfiguration) WithClientCert(value string) *TLSSecretApplyConfiguration {
	b.ClientCert = &value
	return b
}

// WithClientKey sets the ClientKey field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClientKey field is set to the value of the last call.
func (b *TLSSecretApplyConfiguration) WithClientKey(value string) *TLSSecretApplyConfiguration {
	b.ClientKey = &value
	return b
}

// WithRootCAs sets the RootCAs field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RootCAs field is set to the value of the last call.
func (b *TLSSecretApplyConfiguration) WithRootCAs(value string) *TLSSecretApplyConfiguration {
	b.RootCAs = &value
	return b
}

// WithSecret sets the Secret field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Secret field is set to the value of the last call.
func (b *TLSSecretApplyConfiguration) WithSecret(value *SecretRefApplyConfiguration) *TLSSecretApplyConfiguration {
	b.Secret = value
	return b
}
