package v1beta2

import (
	k8sapi "k8s.io/api/core/v1"
)

type CredentialsSecret struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type Status struct {
	ObservedGeneration int64       `json:"observedGeneration"`
	Conditions         []Condition `json:"conditions"`
}

type Condition struct {
	Type               string                 `json:"type"`
	Status             k8sapi.ConditionStatus `json:"status"`
	Reason             string                 `json:"reason"`
	Message            string                 `json:"message"`
	LastTransitionTime string                 `json:"lastTransitionTime"`
}

type BaseStreamConfig struct {
	PreventDelete bool `json:"preventDelete"`
	PreventUpdate bool `json:"preventUpdate"`
	ConnectionOpts
}

type ConnectionOpts struct {
	Account  string   `json:"account,omitempty"`
	Creds    string   `json:"creds,omitempty"`
	Nkey     string   `json:"nkey,omitempty"`
	Servers  []string `json:"servers,omitempty"`
	TLS      TLS      `json:"tls"`
	TLSFirst bool     `json:"tlsFirst,omitempty"`
	JsDomain string   `json:"jsDomain,omitempty"`
}

type ConsumerLimits struct {
	InactiveThreshold string `json:"inactiveThreshold,omitempty"`
	MaxAckPending     int    `json:"maxAckPending,omitempty"`
}

type TLS struct {
	ClientCert string   `json:"clientCert"`
	ClientKey  string   `json:"clientKey"`
	RootCAs    []string `json:"rootCas,omitempty"`
}

type TLSSecret struct {
	ClientCert string     `json:"cert,omitempty"`
	ClientKey  string     `json:"key,omitempty"`
	RootCAs    string     `json:"ca,omitempty"`
	Secret     *SecretRef `json:"secret"`
}

type CredsSecret struct {
	File   string     `json:"file,omitempty"`
	Secret *SecretRef `json:"secret"`
}

type TokenSecret struct {
	Token  string    `json:"token,omitempty"`
	Secret SecretRef `json:"secret"`
}

type User struct {
	User     string    `json:"user,omitempty"`
	Password string    `json:"password,omitempty"`
	Secret   SecretRef `json:"secret"`
}

type SecretRef struct {
	Name string `json:"name"`
}
