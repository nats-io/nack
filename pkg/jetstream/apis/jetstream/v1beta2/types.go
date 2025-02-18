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
	Account  string   `json:"account"`
	Creds    string   `json:"creds"`
	Nkey     string   `json:"nkey"`
	Servers  []string `json:"servers"`
	TLS      TLS      `json:"tls"`
	TLSFirst bool     `json:"tlsFirst"`
}

type ConsumerLimits struct {
	InactiveThreshold string `json:"inactiveThreshold"`
	MaxAckPending     int    `json:"maxAckPending"`
}

type TLS struct {
	ClientCert string   `json:"clientCert"`
	ClientKey  string   `json:"clientKey"`
	RootCAs    []string `json:"rootCas"`
}

type TLSSecret struct {
	ClientCert string     `json:"cert"`
	ClientKey  string     `json:"key"`
	RootCAs    string     `json:"ca"`
	Secret     *SecretRef `json:"secret"`
}

type CredsSecret struct {
	File   string     `json:"file"`
	Secret *SecretRef `json:"secret"`
}

type TokenSecret struct {
	Token  string    `json:"token"`
	Secret SecretRef `json:"secret"`
}

type User struct {
	User     string    `json:"user"`
	Password string    `json:"password"`
	Secret   SecretRef `json:"secret"`
}

type SecretRef struct {
	Name string `json:"name"`
}
