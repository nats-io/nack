package advisory

import (
	"github.com/nats-io/jsm.go/api/event"
)

// JetStreamAPIAuditV1 is a advisory published for any JetStream API access
//
// NATS Schema Type io.nats.jetstream.advisory.v1.api_audit
type JetStreamAPIAuditV1 struct {
	event.NATSEvent

	Server   string           `json:"server"`
	Client   APIAuditClientV1 `json:"client"`
	Subject  string           `json:"subject"`
	Request  string           `json:"request,omitempty"`
	Response string           `json:"response"`
}

type APIAuditClientV1 struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	CID     uint64 `json:"cid"`
	Account string `json:"account"`
	User    string `json:"user,omitempty"`
	Name    string `json:"name,omitempty"`
	Lang    string `json:"lang,omitempty"`
	Version string `json:"version,omitempty"`
}

func init() {
	err := event.RegisterTextCompactTemplate("io.nats.jetstream.advisory.v1.api_audit", `{{ .Time | ShortTime }} [JS API] {{ .Subject }}{{ if .Client.User }} {{ .Client.User}} @{{ end }}{{ if .Client.Account }} {{ .Client.Account }}{{ end }}`)
	if err != nil {
		panic(err)
	}

	err = event.RegisterTextExtendedTemplate("io.nats.jetstream.advisory.v1.api_audit", `
[{{ .Time | ShortTime }}] [{{ .ID }}] JetStream API Access

      Server: {{ .Server }}
     Subject: {{ .Subject }}
      Client:
{{- if .Client.User }}
                      User: {{ .Client.User }} Account: {{ .Client.Account }}
{{- end }}
                      Host: {{ HostPort .Client.Host .Client.Port }}
                       CID: {{ .Client.CID }}
{{- if .Client.Name }}
                      Name: {{ .Client.Name }}
{{- end }}
           Library Version: {{ .Client.Version }}  Language: {{ with .Client.Lang }}{{ . }}{{ else }}Unknown{{ end }}

    Request:
{{ if .Request }}
{{ .Request | LeftPad 10 }}
{{- else }}
          Empty Request
{{- end }}

    Response:

{{ .Response | LeftPad 10 }}`)
	if err != nil {
		panic(err)
	}
}
