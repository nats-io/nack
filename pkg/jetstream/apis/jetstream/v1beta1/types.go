package v1beta1

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

type TLS struct {
	ClientCert string   `json:"clientCert"`
	ClientKey  string   `json:"clientKey"`
	RootCAs    []string `json:"rootCas"`
}
