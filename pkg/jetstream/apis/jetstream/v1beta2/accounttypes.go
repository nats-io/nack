package v1beta2

import (
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Account is a specification for a Account resource.
type Account struct {
	k8smeta.TypeMeta   `json:",inline"`
	k8smeta.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccountSpec `json:"spec"`
	Status Status      `json:"status"`
}

func (c *Account) GetSpec() interface{} {
	return c.Spec
}

// AccountSpec is the spec for a Account resource
type AccountSpec struct {
	Servers []string     `json:"servers"`
	TLS     *TLSSecret   `json:"tls"`
	Creds   *CredsSecret `json:"creds"`
	Token   *TokenSecret `json:"token"`
	User    *User        `json:"user"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccountList is a list of Account resources
type AccountList struct {
	k8smeta.TypeMeta `json:",inline"`
	k8smeta.ListMeta `json:"metadata"`

	Items []Account `json:"items"`
}
