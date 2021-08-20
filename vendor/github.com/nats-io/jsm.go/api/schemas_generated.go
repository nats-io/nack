// auto generated 2021-07-19 09:35:47.980711 +0200 CEST m=+0.028163459

package api

import (
	jsadvisory "github.com/nats-io/jsm.go/api/jetstream/advisory"
	jsmetric "github.com/nats-io/jsm.go/api/jetstream/metric"
	srvadvisory "github.com/nats-io/jsm.go/api/server/advisory"
	srvmetric "github.com/nats-io/jsm.go/api/server/metric"
	scfs "github.com/nats-io/jsm.go/schemas"
)

var schemaTypes = map[string]func() interface{}{
	"io.nats.server.advisory.v1.client_connect":                  func() interface{} { return &srvadvisory.ConnectEventMsgV1{} },
	"io.nats.server.advisory.v1.client_disconnect":               func() interface{} { return &srvadvisory.DisconnectEventMsgV1{} },
	"io.nats.server.advisory.v1.account_connections":             func() interface{} { return &srvadvisory.AccountConnectionsV1{} },
	"io.nats.server.metric.v1.service_latency":                   func() interface{} { return &srvmetric.ServiceLatencyV1{} },
	"io.nats.jetstream.advisory.v1.api_audit":                    func() interface{} { return &jsadvisory.JetStreamAPIAuditV1{} },
	"io.nats.jetstream.advisory.v1.max_deliver":                  func() interface{} { return &jsadvisory.ConsumerDeliveryExceededAdvisoryV1{} },
	"io.nats.jetstream.advisory.v1.terminated":                   func() interface{} { return &jsadvisory.JSConsumerDeliveryTerminatedAdvisoryV1{} },
	"io.nats.jetstream.advisory.v1.stream_action":                func() interface{} { return &jsadvisory.JSStreamActionAdvisoryV1{} },
	"io.nats.jetstream.advisory.v1.consumer_action":              func() interface{} { return &jsadvisory.JSConsumerActionAdvisoryV1{} },
	"io.nats.jetstream.advisory.v1.snapshot_create":              func() interface{} { return &jsadvisory.JSSnapshotCreateAdvisoryV1{} },
	"io.nats.jetstream.advisory.v1.snapshot_complete":            func() interface{} { return &jsadvisory.JSSnapshotCompleteAdvisoryV1{} },
	"io.nats.jetstream.advisory.v1.restore_create":               func() interface{} { return &jsadvisory.JSRestoreCreateAdvisoryV1{} },
	"io.nats.jetstream.advisory.v1.restore_complete":             func() interface{} { return &jsadvisory.JSRestoreCompleteAdvisoryV1{} },
	"io.nats.jetstream.advisory.v1.stream_leader_elected":        func() interface{} { return &jsadvisory.JSStreamLeaderElectedV1{} },
	"io.nats.jetstream.advisory.v1.consumer_leader_elected":      func() interface{} { return &jsadvisory.JSConsumerLeaderElectedV1{} },
	"io.nats.jetstream.advisory.v1.stream_quorum_lost":           func() interface{} { return &jsadvisory.JSStreamQuorumLostV1{} },
	"io.nats.jetstream.advisory.v1.consumer_quorum_lost":         func() interface{} { return &jsadvisory.JSConsumerQuorumLostV1{} },
	"io.nats.jetstream.advisory.v1.server_out_of_space":          func() interface{} { return &jsadvisory.JSServerOutOfSpaceAdvisoryV1{} },
	"io.nats.jetstream.metric.v1.consumer_ack":                   func() interface{} { return &jsmetric.ConsumerAckMetricV1{} },
	"io.nats.jetstream.api.v1.consumer_configuration":            func() interface{} { return &ConsumerConfig{} },
	"io.nats.jetstream.api.v1.stream_configuration":              func() interface{} { return &StreamConfig{} },
	"io.nats.jetstream.api.v1.stream_template_configuration":     func() interface{} { return &StreamTemplateConfig{} },
	"io.nats.jetstream.api.v1.account_info_response":             func() interface{} { return &JSApiAccountInfoResponse{} },
	"io.nats.jetstream.api.v1.consumer_create_request":           func() interface{} { return &JSApiConsumerCreateRequest{} },
	"io.nats.jetstream.api.v1.consumer_create_response":          func() interface{} { return &JSApiConsumerCreateResponse{} },
	"io.nats.jetstream.api.v1.consumer_delete_response":          func() interface{} { return &JSApiConsumerDeleteResponse{} },
	"io.nats.jetstream.api.v1.consumer_info_response":            func() interface{} { return &JSApiConsumerInfoResponse{} },
	"io.nats.jetstream.api.v1.consumer_list_request":             func() interface{} { return &JSApiConsumerListRequest{} },
	"io.nats.jetstream.api.v1.consumer_list_response":            func() interface{} { return &JSApiConsumerListResponse{} },
	"io.nats.jetstream.api.v1.consumer_names_request":            func() interface{} { return &JSApiConsumerNamesRequest{} },
	"io.nats.jetstream.api.v1.consumer_names_response":           func() interface{} { return &JSApiConsumerNamesResponse{} },
	"io.nats.jetstream.api.v1.consumer_getnext_request":          func() interface{} { return &JSApiConsumerGetNextRequest{} },
	"io.nats.jetstream.api.v1.consumer_leader_stepdown_response": func() interface{} { return &JSApiConsumerLeaderStepDownResponse{} },
	"io.nats.jetstream.api.v1.stream_create_request":             func() interface{} { return &JSApiStreamCreateRequest{} },
	"io.nats.jetstream.api.v1.stream_create_response":            func() interface{} { return &JSApiStreamCreateResponse{} },
	"io.nats.jetstream.api.v1.stream_delete_response":            func() interface{} { return &JSApiStreamDeleteResponse{} },
	"io.nats.jetstream.api.v1.stream_info_request":               func() interface{} { return &JSApiStreamInfoRequest{} },
	"io.nats.jetstream.api.v1.stream_info_response":              func() interface{} { return &JSApiStreamInfoResponse{} },
	"io.nats.jetstream.api.v1.stream_list_request":               func() interface{} { return &JSApiStreamListRequest{} },
	"io.nats.jetstream.api.v1.stream_list_response":              func() interface{} { return &JSApiStreamListResponse{} },
	"io.nats.jetstream.api.v1.stream_msg_delete_response":        func() interface{} { return &JSApiMsgDeleteResponse{} },
	"io.nats.jetstream.api.v1.stream_msg_get_request":            func() interface{} { return &JSApiMsgGetRequest{} },
	"io.nats.jetstream.api.v1.stream_msg_get_response":           func() interface{} { return &JSApiMsgGetResponse{} },
	"io.nats.jetstream.api.v1.stream_names_request":              func() interface{} { return &JSApiStreamNamesRequest{} },
	"io.nats.jetstream.api.v1.stream_names_response":             func() interface{} { return &JSApiStreamNamesResponse{} },
	"io.nats.jetstream.api.v1.stream_purge_response":             func() interface{} { return &JSApiStreamPurgeResponse{} },
	"io.nats.jetstream.api.v1.stream_snapshot_response":          func() interface{} { return &JSApiStreamSnapshotResponse{} },
	"io.nats.jetstream.api.v1.stream_snapshot_request":           func() interface{} { return &JSApiStreamSnapshotRequest{} },
	"io.nats.jetstream.api.v1.stream_restore_request":            func() interface{} { return &JSApiStreamRestoreRequest{} },
	"io.nats.jetstream.api.v1.stream_restore_response":           func() interface{} { return &JSApiStreamRestoreResponse{} },
	"io.nats.jetstream.api.v1.stream_template_create_request":    func() interface{} { return &JSApiStreamTemplateCreateRequest{} },
	"io.nats.jetstream.api.v1.stream_template_create_response":   func() interface{} { return &JSApiStreamTemplateCreateResponse{} },
	"io.nats.jetstream.api.v1.stream_template_delete_response":   func() interface{} { return &JSApiStreamTemplateDeleteResponse{} },
	"io.nats.jetstream.api.v1.stream_template_info_response":     func() interface{} { return &JSApiStreamTemplateInfoResponse{} },
	"io.nats.jetstream.api.v1.stream_template_names_response":    func() interface{} { return &JSApiStreamTemplateNamesResponse{} },
	"io.nats.jetstream.api.v1.stream_template_names_request":     func() interface{} { return &JSApiStreamTemplateNamesRequest{} },
	"io.nats.jetstream.api.v1.stream_update_response":            func() interface{} { return &JSApiStreamUpdateResponse{} },
	"io.nats.jetstream.api.v1.stream_remove_peer_request":        func() interface{} { return &JSApiStreamRemovePeerRequest{} },
	"io.nats.jetstream.api.v1.stream_remove_peer_response":       func() interface{} { return &JSApiStreamRemovePeerResponse{} },
	"io.nats.jetstream.api.v1.stream_leader_stepdown_response":   func() interface{} { return &JSApiStreamLeaderStepDownResponse{} },
	"io.nats.jetstream.api.v1.pub_ack_response":                  func() interface{} { return &JSPubAckResponse{} },
	"io.nats.jetstream.api.v1.meta_leader_stepdown_request":      func() interface{} { return &JSApiLeaderStepDownRequest{} },
	"io.nats.jetstream.api.v1.meta_leader_stepdown_response":     func() interface{} { return &JSApiLeaderStepDownResponse{} },
	"io.nats.jetstream.api.v1.meta_server_remove_request":        func() interface{} { return &JSApiMetaServerRemoveRequest{} },
	"io.nats.jetstream.api.v1.meta_server_remove_response":       func() interface{} { return &JSApiMetaServerRemoveResponse{} },
	"io.nats.unknown_message":                                    func() interface{} { return &UnknownMessage{} },
}

// Validate performs a JSON Schema validation of the configuration
func (t ConsumerConfig) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_configuration
func (t ConsumerConfig) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_configuration"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t ConsumerConfig) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_configuration.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t ConsumerConfig) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t StreamConfig) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_configuration
func (t StreamConfig) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_configuration"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t StreamConfig) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_configuration.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t StreamConfig) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t StreamTemplateConfig) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_template_configuration
func (t StreamTemplateConfig) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_template_configuration"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t StreamTemplateConfig) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_template_configuration.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t StreamTemplateConfig) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiAccountInfoResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.account_info_response
func (t JSApiAccountInfoResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.account_info_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiAccountInfoResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/account_info_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiAccountInfoResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiConsumerCreateRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_create_request
func (t JSApiConsumerCreateRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_create_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiConsumerCreateRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_create_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiConsumerCreateRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiConsumerCreateResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_create_response
func (t JSApiConsumerCreateResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_create_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiConsumerCreateResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_create_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiConsumerCreateResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiConsumerDeleteResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_delete_response
func (t JSApiConsumerDeleteResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_delete_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiConsumerDeleteResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_delete_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiConsumerDeleteResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiConsumerInfoResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_info_response
func (t JSApiConsumerInfoResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_info_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiConsumerInfoResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_info_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiConsumerInfoResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiConsumerListRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_list_request
func (t JSApiConsumerListRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_list_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiConsumerListRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_list_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiConsumerListRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiConsumerListResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_list_response
func (t JSApiConsumerListResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_list_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiConsumerListResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_list_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiConsumerListResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiConsumerNamesRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_names_request
func (t JSApiConsumerNamesRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_names_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiConsumerNamesRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_names_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiConsumerNamesRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiConsumerNamesResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_names_response
func (t JSApiConsumerNamesResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_names_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiConsumerNamesResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_names_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiConsumerNamesResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiConsumerGetNextRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_getnext_request
func (t JSApiConsumerGetNextRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_getnext_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiConsumerGetNextRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_getnext_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiConsumerGetNextRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiConsumerLeaderStepDownResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.consumer_leader_stepdown_response
func (t JSApiConsumerLeaderStepDownResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_leader_stepdown_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiConsumerLeaderStepDownResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/consumer_leader_stepdown_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiConsumerLeaderStepDownResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamCreateRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_create_request
func (t JSApiStreamCreateRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_create_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamCreateRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_create_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamCreateRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamCreateResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_create_response
func (t JSApiStreamCreateResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_create_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamCreateResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_create_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamCreateResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamDeleteResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_delete_response
func (t JSApiStreamDeleteResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_delete_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamDeleteResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_delete_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamDeleteResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamInfoRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_info_request
func (t JSApiStreamInfoRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_info_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamInfoRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_info_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamInfoRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamInfoResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_info_response
func (t JSApiStreamInfoResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_info_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamInfoResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_info_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamInfoResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamListRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_list_request
func (t JSApiStreamListRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_list_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamListRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_list_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamListRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamListResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_list_response
func (t JSApiStreamListResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_list_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamListResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_list_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamListResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiMsgDeleteResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_msg_delete_response
func (t JSApiMsgDeleteResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_msg_delete_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiMsgDeleteResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_msg_delete_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiMsgDeleteResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiMsgGetRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_msg_get_request
func (t JSApiMsgGetRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_msg_get_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiMsgGetRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_msg_get_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiMsgGetRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiMsgGetResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_msg_get_response
func (t JSApiMsgGetResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_msg_get_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiMsgGetResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_msg_get_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiMsgGetResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamNamesRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_names_request
func (t JSApiStreamNamesRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_names_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamNamesRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_names_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamNamesRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamNamesResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_names_response
func (t JSApiStreamNamesResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_names_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamNamesResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_names_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamNamesResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamPurgeResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_purge_response
func (t JSApiStreamPurgeResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_purge_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamPurgeResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_purge_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamPurgeResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamSnapshotResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_snapshot_response
func (t JSApiStreamSnapshotResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_snapshot_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamSnapshotResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_snapshot_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamSnapshotResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamSnapshotRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_snapshot_request
func (t JSApiStreamSnapshotRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_snapshot_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamSnapshotRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_snapshot_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamSnapshotRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamRestoreRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_restore_request
func (t JSApiStreamRestoreRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_restore_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamRestoreRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_restore_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamRestoreRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamRestoreResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_restore_response
func (t JSApiStreamRestoreResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_restore_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamRestoreResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_restore_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamRestoreResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamTemplateCreateRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_template_create_request
func (t JSApiStreamTemplateCreateRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_template_create_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamTemplateCreateRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_template_create_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamTemplateCreateRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamTemplateCreateResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_template_create_response
func (t JSApiStreamTemplateCreateResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_template_create_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamTemplateCreateResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_template_create_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamTemplateCreateResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamTemplateDeleteResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_template_delete_response
func (t JSApiStreamTemplateDeleteResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_template_delete_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamTemplateDeleteResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_template_delete_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamTemplateDeleteResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamTemplateInfoResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_template_info_response
func (t JSApiStreamTemplateInfoResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_template_info_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamTemplateInfoResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_template_info_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamTemplateInfoResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamTemplateNamesResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_template_names_response
func (t JSApiStreamTemplateNamesResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_template_names_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamTemplateNamesResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_template_names_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamTemplateNamesResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamTemplateNamesRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_template_names_request
func (t JSApiStreamTemplateNamesRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_template_names_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamTemplateNamesRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_template_names_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamTemplateNamesRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamUpdateResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_update_response
func (t JSApiStreamUpdateResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_update_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamUpdateResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_update_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamUpdateResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamRemovePeerRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_remove_peer_request
func (t JSApiStreamRemovePeerRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_remove_peer_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamRemovePeerRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_remove_peer_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamRemovePeerRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamRemovePeerResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_remove_peer_response
func (t JSApiStreamRemovePeerResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_remove_peer_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamRemovePeerResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_remove_peer_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamRemovePeerResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiStreamLeaderStepDownResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.stream_leader_stepdown_response
func (t JSApiStreamLeaderStepDownResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.stream_leader_stepdown_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiStreamLeaderStepDownResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/stream_leader_stepdown_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiStreamLeaderStepDownResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSPubAckResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.pub_ack_response
func (t JSPubAckResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.pub_ack_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSPubAckResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/pub_ack_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSPubAckResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiLeaderStepDownRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.meta_leader_stepdown_request
func (t JSApiLeaderStepDownRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.meta_leader_stepdown_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiLeaderStepDownRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/meta_leader_stepdown_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiLeaderStepDownRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiLeaderStepDownResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.meta_leader_stepdown_response
func (t JSApiLeaderStepDownResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.meta_leader_stepdown_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiLeaderStepDownResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/meta_leader_stepdown_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiLeaderStepDownResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiMetaServerRemoveRequest) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.meta_server_remove_request
func (t JSApiMetaServerRemoveRequest) SchemaType() string {
	return "io.nats.jetstream.api.v1.meta_server_remove_request"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiMetaServerRemoveRequest) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/meta_server_remove_request.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiMetaServerRemoveRequest) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}

// Validate performs a JSON Schema validation of the configuration
func (t JSApiMetaServerRemoveResponse) Validate(v ...StructValidator) (valid bool, errors []string) {
	if len(v) == 0 || v[0] == nil {
		return true, nil
	}

	return v[0].ValidateStruct(t, t.SchemaType())
}

// SchemaType is the NATS schema type io.nats.jetstream.api.v1.meta_server_remove_response
func (t JSApiMetaServerRemoveResponse) SchemaType() string {
	return "io.nats.jetstream.api.v1.meta_server_remove_response"
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (t JSApiMetaServerRemoveResponse) SchemaID() string {
	return "https://raw.githubusercontent.com/nats-io/jsm.go/master/schemas/jetstream/api/v1/meta_server_remove_response.json"
}

// Schema is a JSON Schema document for the JetStream Consumer Configuration
func (t JSApiMetaServerRemoveResponse) Schema() ([]byte, error) {
	f, err := SchemaFileForType(t.SchemaType())
	if err != nil {
		return nil, err
	}
	return scfs.Load(f)
}
