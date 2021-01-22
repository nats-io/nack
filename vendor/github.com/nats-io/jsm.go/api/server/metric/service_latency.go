package metric

import (
	"net/http"
	"time"

	"github.com/nats-io/jsm.go/api/event"
)

// ServiceLatencyV1 is the JSON message sent out in response to latency tracking for
// exported services.
//
// NATS Schema Type io.nats.server.metric.v1.service_latency
type ServiceLatencyV1 struct {
	event.NATSEvent

	Status         int             `json:"status"`
	Error          string          `json:"description,omitempty"`
	Requestor      LatencyClientV1 `json:"requestor,omitempty"`
	Responder      LatencyClientV1 `json:"responder,omitempty"`
	RequestHeader  http.Header     `json:"header,omitempty"`
	RequestStart   time.Time       `json:"start"`
	ServiceLatency time.Duration   `json:"service"`
	SystemLatency  time.Duration   `json:"system"`
	TotalLatency   time.Duration   `json:"total"`
}

type LatencyClientV1 struct {
	Account string        `json:"acc"`
	RTT     time.Duration `json:"rtt"`
	Start   time.Time     `json:"start,omitempty"`
	User    string        `json:"user,omitempty"`
	Name    string        `json:"name,omitempty"`
	Lang    string        `json:"lang,omitempty"`
	Version string        `json:"ver,omitempty"`
	IP      string        `json:"ip,omitempty"`
	CID     uint64        `json:"cid,omitempty"`
	Server  string        `json:"server,omitempty"`
}

func init() {
	err := event.RegisterTextCompactTemplate("io.nats.server.metric.v1.service_latency", `{{ .Time | ShortTime }} [Svc Latency] {{ if .Error }}{{ .Error }} {{ end }}requestor {{ .Requestor.RTT }} <-> system {{ .SystemLatency }} <- service rtt {{ .Responder.RTT }} -> service {{ .ServiceLatency }}`)
	if err != nil {
		panic(err)
	}

	err = event.RegisterTextExtendedTemplate("io.nats.server.metric.v1.service_latency", `
{{- if .Error }}
[{{ .Time | ShortTime }}] [{{ .ID }}] Service Latency - {{ .Error }}
{{- else }}
[{{ .Time | ShortTime }}] [{{ .ID }}] Service Latency
{{- end }}

   Start Time: {{ .RequestStart | NanoTime }}
{{- if .Error }}
        Error: {{ .Error }}
{{- end }}

   Latencies:

      Request Duration: {{ .TotalLatency }}
{{- if .Requestor }}
             Requestor: {{ .Requestor.RTT }}
{{- end }}
           NATS System: {{ .SystemLatency }}
               Service: {{ .ServiceLatency }}
{{ with .Requestor }}
   Requestor:
     Account: {{ .Account }}
         RTT: {{ .RTT }}
{{- if .User }}
       Start: {{ .Start }}
        User: {{ .User }}
        Name: {{ .Name }}
    Language: {{ .Lang }}
     Version: {{ .Version }}
{{- end }}
{{- if .CID }}
          IP: {{ .IP }}
         CID: {{ .CID }}
      Server: {{ .Server }}
{{- end }}
{{- end }}
{{ with .Responder }}
   Responder:
     Account: {{ .Account }}
         RTT: {{ .RTT }}
{{- if .User }}
       Start: {{ .Start }}
        User: {{ .User }}
        Name: {{ .Name }}
    Language: {{ .Lang }}
     Version: {{ .Version }}
{{- end }}
{{- if .CID }}
          IP: {{ .IP }}
         CID: {{ .CID }}
      Server: {{ .Server }}
{{- end }}
{{- end }}`)
	if err != nil {
		panic(err)
	}
}
