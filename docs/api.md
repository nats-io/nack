# API Reference

Packages:

- [jetstream.nats.io/v1beta2](#jetstreamnatsiov1beta2)
- [jetstream.nats.io/v1beta1](#jetstreamnatsiov1beta1)

# jetstream.nats.io/v1beta2

Resource Types:

- [Stream](#stream)

- [Consumer](#consumer)

- [Account](#account)

- [KeyValue](#keyvalue)

- [ObjectStore](#objectstore)

## Stream

<sup><sup>[↩ Parent](#jetstreamnatsiov1beta2)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>jetstream.nats.io/v1beta2</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Stream</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#streamspec">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamstatus">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec

<sup><sup>[↩ Parent](#stream)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>account</b></td>
        <td>string</td>
        <td>
          Name of the account to which the Stream belongs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>allowDirect</b></td>
        <td>boolean</td>
        <td>
          When true, allow higher performance, direct access to get individual messages.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>allowRollup</b></td>
        <td>boolean</td>
        <td>
          When true, allows the use of the Nats-Rollup header to replace all contents of a stream, or subject in a stream, with a single new message.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>compression</b></td>
        <td>enum</td>
        <td>
          Stream specific compression.<br/>
          <br/>
            <i>Enum</i>: s2, none, <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecconsumerlimits">consumerLimits</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>creds</b></td>
        <td>string</td>
        <td>
          NATS user credentials for connecting to servers. Please make sure your controller has mounted the creds on this path.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>denyDelete</b></td>
        <td>boolean</td>
        <td>
          When true, restricts the ability to delete messages from a stream via the API. Cannot be changed once set to true.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>denyPurge</b></td>
        <td>boolean</td>
        <td>
          When true, restricts the ability to purge a stream via the API. Cannot be changed once set to true.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          The description of the stream.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>discard</b></td>
        <td>enum</td>
        <td>
          When a Stream reach it's limits either old messages are deleted or new ones are denied.<br/>
          <br/>
            <i>Enum</i>: old, new<br/>
            <i>Default</i>: old<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>discardPerSubject</b></td>
        <td>boolean</td>
        <td>
          Applies discard policy on a per-subject basis. Requires discard policy 'new' and 'maxMsgs' to be set.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>duplicateWindow</b></td>
        <td>string</td>
        <td>
          The duration window to track duplicate messages for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>firstSequence</b></td>
        <td>number</td>
        <td>
          Sequence number from which the Stream will start.<br/>
          <br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxAge</b></td>
        <td>string</td>
        <td>
          Maximum age of any message in the stream, expressed in Go's time.Duration format. Empty for unlimited.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxBytes</b></td>
        <td>integer</td>
        <td>
          How big the Stream may be, when the combined stream size exceeds this old messages are removed. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxConsumers</b></td>
        <td>integer</td>
        <td>
          How many Consumers can be defined for a given Stream. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxMsgSize</b></td>
        <td>integer</td>
        <td>
          The largest message that will be accepted by the Stream. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxMsgs</b></td>
        <td>integer</td>
        <td>
          How many messages may be in a Stream, oldest messages will be removed if the Stream exceeds this size. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxMsgsPerSubject</b></td>
        <td>integer</td>
        <td>
          The maximum number of messages per subject.<br/>
          <br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>allowMsgTtl</b></td>
        <td>boolean</td>
        <td>
          When true, allows header initiated per-message TTLs. If disabled, then the `NATS-TTL` header will be ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subjectDeleteMarkerTtl</b></td>
        <td>string</td>
        <td>
          Enables and sets a duration for adding server markers for delete, purge and max age limits, expressed in Go's time.Duration format.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>metadata</b></td>
        <td>map[string]string</td>
        <td>
          Additional Stream metadata.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecmirror">mirror</a></b></td>
        <td>object</td>
        <td>
          A stream mirror.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mirrorDirect</b></td>
        <td>boolean</td>
        <td>
          When true, enables direct access to messages from the origin stream.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          A unique name for the Stream.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nkey</b></td>
        <td>string</td>
        <td>
          NATS user NKey for connecting to servers.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>noAck</b></td>
        <td>boolean</td>
        <td>
          Disables acknowledging messages that are received by the Stream.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecplacement">placement</a></b></td>
        <td>object</td>
        <td>
          A stream's placement.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>preventDelete</b></td>
        <td>boolean</td>
        <td>
          When true, the managed Stream will not be deleted when the resource is deleted.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>preventUpdate</b></td>
        <td>boolean</td>
        <td>
          When true, the managed Stream will not be updated when the resource is updated.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          How many replicas to keep for each message.<br/>
          <br/>
            <i>Default</i>: 1<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecrepublish">republish</a></b></td>
        <td>object</td>
        <td>
          Republish configuration of the stream.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>retention</b></td>
        <td>enum</td>
        <td>
          How messages are retained in the Stream, once this is exceeded old messages are removed.<br/>
          <br/>
            <i>Enum</i>: limits, interest, workqueue<br/>
            <i>Default</i>: limits<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sealed</b></td>
        <td>boolean</td>
        <td>
          Seal an existing stream so no new messages may be added.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>servers</b></td>
        <td>[]string</td>
        <td>
          A list of servers for creating stream.<br/>
          <br/>
            <i>Default</i>: []<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecsourcesindex">sources</a></b></td>
        <td>[]object</td>
        <td>
          A stream's sources.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storage</b></td>
        <td>enum</td>
        <td>
          The storage backend to use for the Stream.<br/>
          <br/>
            <i>Enum</i>: file, memory<br/>
            <i>Default</i>: memory<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecsubjecttransform">subjectTransform</a></b></td>
        <td>object</td>
        <td>
          SubjectTransform is for applying a subject transform (to matching messages) when a new message is received.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subjects</b></td>
        <td>[]string</td>
        <td>
          A list of subjects to consume, supports wildcards.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspectls">tls</a></b></td>
        <td>object</td>
        <td>
          A client's TLS certs and keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tlsFirst</b></td>
        <td>boolean</td>
        <td>
          When true, the KV Store will initiate TLS before server INFO.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.consumerLimits

<sup><sup>[↩ Parent](#streamspec)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>inactiveThreshold</b></td>
        <td>string</td>
        <td>
          The duration of inactivity after which a consumer is considered inactive.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxAckPending</b></td>
        <td>integer</td>
        <td>
          Maximum number of outstanding unacknowledged messages.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.mirror

<sup><sup>[↩ Parent](#streamspec)</sup></sup>

A stream mirror.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>externalApiPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>externalDeliverPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>filterSubject</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartSeq</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartTime</b></td>
        <td>string</td>
        <td>
          Time format must be RFC3339.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecmirrorsubjecttransformsindex">subjectTransforms</a></b></td>
        <td>[]object</td>
        <td>
          List of subject transforms for this mirror.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.mirror.subjectTransforms[index]

<sup><sup>[↩ Parent](#streamspecmirror)</sup></sup>

A subject transform pair.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>dest</b></td>
        <td>string</td>
        <td>
          Destination subject.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>source</b></td>
        <td>string</td>
        <td>
          Source subject.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.placement

<sup><sup>[↩ Parent](#streamspec)</sup></sup>

A stream's placement.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cluster</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tags</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.republish

<sup><sup>[↩ Parent](#streamspec)</sup></sup>

Republish configuration of the stream.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>destination</b></td>
        <td>string</td>
        <td>
          Messages will be additionally published to this subject.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>source</b></td>
        <td>string</td>
        <td>
          Messages will be published from this subject to the destination subject.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.sources[index]

<sup><sup>[↩ Parent](#streamspec)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>externalApiPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>externalDeliverPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>filterSubject</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartSeq</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartTime</b></td>
        <td>string</td>
        <td>
          Time format must be RFC3339.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecsourcesindexsubjecttransformsindex">subjectTransforms</a></b></td>
        <td>[]object</td>
        <td>
          List of subject transforms for this mirror.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.sources[index].subjectTransforms[index]

<sup><sup>[↩ Parent](#streamspecsourcesindex)</sup></sup>

A subject transform pair.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>dest</b></td>
        <td>string</td>
        <td>
          Destination subject.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>source</b></td>
        <td>string</td>
        <td>
          Source subject.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.subjectTransform

<sup><sup>[↩ Parent](#streamspec)</sup></sup>

SubjectTransform is for applying a subject transform (to matching messages) when a new message is received.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>dest</b></td>
        <td>string</td>
        <td>
          Destination subject to transform into.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>source</b></td>
        <td>string</td>
        <td>
          Source subject.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.tls

<sup><sup>[↩ Parent](#streamspec)</sup></sup>

A client's TLS certs and keys.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientCert</b></td>
        <td>string</td>
        <td>
          A client's cert filepath. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clientKey</b></td>
        <td>string</td>
        <td>
          A client's key filepath. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>rootCas</b></td>
        <td>[]string</td>
        <td>
          A list of filepaths to CAs. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.status

<sup><sup>[↩ Parent](#stream)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#streamstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.status.conditions[index]

<sup><sup>[↩ Parent](#streamstatus)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Consumer

<sup><sup>[↩ Parent](#jetstreamnatsiov1beta2)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>jetstream.nats.io/v1beta2</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Consumer</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#consumerspec">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#consumerstatus">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Consumer.spec

<sup><sup>[↩ Parent](#consumer)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>account</b></td>
        <td>string</td>
        <td>
          Name of the account to which the Consumer belongs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ackPolicy</b></td>
        <td>enum</td>
        <td>
          How messages should be acknowledged.<br/>
          <br/>
            <i>Enum</i>: none, all, explicit<br/>
            <i>Default</i>: none<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ackWait</b></td>
        <td>string</td>
        <td>
          How long to allow messages to remain un-acknowledged before attempting redelivery.<br/>
          <br/>
            <i>Default</i>: 1ns<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>backoff</b></td>
        <td>[]string</td>
        <td>
          List of durations representing a retry time scale for NaK'd or retried messages.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>creds</b></td>
        <td>string</td>
        <td>
          NATS user credentials for connecting to servers. Please make sure your controller has mounted the creds on its path.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>deliverGroup</b></td>
        <td>string</td>
        <td>
          The name of a queue group.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>deliverPolicy</b></td>
        <td>enum</td>
        <td>
            <i>Enum</i>: all, last, new, byStartSequence, byStartTime<br/>
            <i>Default</i>: all<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>deliverSubject</b></td>
        <td>string</td>
        <td>
          The subject to deliver observed messages, when not set, a pull-based Consumer is created.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          The description of the consumer.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>durableName</b></td>
        <td>string</td>
        <td>
          The name of the Consumer.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>filterSubject</b></td>
        <td>string</td>
        <td>
          Select only a specific incoming subjects, supports wildcards.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>filterSubjects</b></td>
        <td>[]string</td>
        <td>
          List of incoming subjects, supports wildcards. Available since 2.10.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>flowControl</b></td>
        <td>boolean</td>
        <td>
          Enables flow control.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>headersOnly</b></td>
        <td>boolean</td>
        <td>
          When set, only the headers of messages in the stream are delivered, and not the bodies. Additionally, Nats-Msg-Size header is added to indicate the size of the removed payload.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>heartbeatInterval</b></td>
        <td>string</td>
        <td>
          The interval used to deliver idle heartbeats for push-based consumers, in Go's time.Duration format.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>inactiveThreshold</b></td>
        <td>string</td>
        <td>
          The idle time an Ephemeral Consumer allows before it is removed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxAckPending</b></td>
        <td>integer</td>
        <td>
          Maximum pending Acks before consumers are paused.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxDeliver</b></td>
        <td>integer</td>
        <td>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxRequestBatch</b></td>
        <td>integer</td>
        <td>
          The largest batch property that may be specified when doing a pull on a Pull Consumer.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxRequestExpires</b></td>
        <td>string</td>
        <td>
          The maximum expires duration that may be set when doing a pull on a Pull Consumer.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxRequestMaxBytes</b></td>
        <td>integer</td>
        <td>
          The maximum max_bytes value that maybe set when dong a pull on a Pull Consumer.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxWaiting</b></td>
        <td>integer</td>
        <td>
          The number of pulls that can be outstanding on a pull consumer, pulls received after this is reached are ignored.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memStorage</b></td>
        <td>boolean</td>
        <td>
          Force the consumer state to be kept in memory rather than inherit the setting from the stream.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>metadata</b></td>
        <td>map[string]string</td>
        <td>
          Additional Consumer metadata.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nkey</b></td>
        <td>string</td>
        <td>
          NATS user NKey for connecting to servers.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartSeq</b></td>
        <td>integer</td>
        <td>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartTime</b></td>
        <td>string</td>
        <td>
          Time format must be RFC3339.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>preventDelete</b></td>
        <td>boolean</td>
        <td>
          When true, the managed Consumer will not be deleted when the resource is deleted.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>preventUpdate</b></td>
        <td>boolean</td>
        <td>
          When true, the managed Consumer will not be updated when the resource is updated.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>rateLimitBps</b></td>
        <td>integer</td>
        <td>
          Rate at which messages will be delivered to clients, expressed in bit per second.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replayPolicy</b></td>
        <td>enum</td>
        <td>
          How messages are sent.<br/>
          <br/>
            <i>Enum</i>: instant, original<br/>
            <i>Default</i>: instant<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          When set do not inherit the replica count from the stream but specifically set it to this amount.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampleFreq</b></td>
        <td>string</td>
        <td>
          What percentage of acknowledgements should be samples for observability.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>servers</b></td>
        <td>[]string</td>
        <td>
          A list of servers for creating consumer.<br/>
          <br/>
            <i>Default</i>: []<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>streamName</b></td>
        <td>string</td>
        <td>
          The name of the Stream to create the Consumer in.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#consumerspectls">tls</a></b></td>
        <td>object</td>
        <td>
          A client's TLS certs and keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tlsFirst</b></td>
        <td>boolean</td>
        <td>
          When true, the KV Store will initiate TLS before server INFO.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Consumer.spec.tls

<sup><sup>[↩ Parent](#consumerspec)</sup></sup>

A client's TLS certs and keys.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientCert</b></td>
        <td>string</td>
        <td>
          A client's cert filepath. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clientKey</b></td>
        <td>string</td>
        <td>
          A client's key filepath. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>rootCas</b></td>
        <td>[]string</td>
        <td>
          A list of filepaths to CAs. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Consumer.status

<sup><sup>[↩ Parent](#consumer)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#consumerstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Consumer.status.conditions[index]

<sup><sup>[↩ Parent](#consumerstatus)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Account

<sup><sup>[↩ Parent](#jetstreamnatsiov1beta2)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>jetstream.nats.io/v1beta2</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Account</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#accountspec">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#accountstatus">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.spec

<sup><sup>[↩ Parent](#account)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#accountspeccreds">creds</a></b></td>
        <td>object</td>
        <td>
          The creds to be used to connect to the NATS Service.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          A unique name for the Account.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>servers</b></td>
        <td>[]string</td>
        <td>
          A list of servers to connect.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#accountspectls">tls</a></b></td>
        <td>object</td>
        <td>
          The TLS certs to be used to connect to the NATS Service.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tlsFirst</b></td>
        <td>boolean</td>
        <td>
          When true, the KV Store will initiate TLS before server INFO.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#accountspectoken">token</a></b></td>
        <td>object</td>
        <td>
          The token to be used to connect to the NATS Service.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#accountspecuser">user</a></b></td>
        <td>object</td>
        <td>
          The user and password to be used to connect to the NATS Service.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.spec.creds

<sup><sup>[↩ Parent](#accountspec)</sup></sup>

The creds to be used to connect to the NATS Service.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>file</b></td>
        <td>string</td>
        <td>
          Credentials file, generated with github.com/nats-io/nsc tool.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#accountspeccredssecret">secret</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.spec.creds.secret

<sup><sup>[↩ Parent](#accountspeccreds)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the secret with the creds.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.spec.tls

<sup><sup>[↩ Parent](#accountspec)</sup></sup>

The TLS certs to be used to connect to the NATS Service.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>ca</b></td>
        <td>string</td>
        <td>
          Filename of the Root CA of the TLS cert.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>cert</b></td>
        <td>string</td>
        <td>
          Filename of the TLS cert.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Filename of the TLS cert key.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#accountspectlssecret">secret</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.spec.tls.secret

<sup><sup>[↩ Parent](#accountspectls)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the TLS secret with the certs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.spec.token

<sup><sup>[↩ Parent](#accountspec)</sup></sup>

The token to be used to connect to the NATS Service.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#accountspectokensecret">secret</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>token</b></td>
        <td>string</td>
        <td>
          Key in the secret that contains the token.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.spec.token.secret

<sup><sup>[↩ Parent](#accountspectoken)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the secret with the token.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.spec.user

<sup><sup>[↩ Parent](#accountspec)</sup></sup>

The user and password to be used to connect to the NATS Service.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>password</b></td>
        <td>string</td>
        <td>
          Key in the secret that contains the password.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#accountspecusersecret">secret</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          Key in the secret that contains the user.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.spec.user.secret

<sup><sup>[↩ Parent](#accountspecuser)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the secret with the user and password.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.status

<sup><sup>[↩ Parent](#account)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#accountstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Account.status.conditions[index]

<sup><sup>[↩ Parent](#accountstatus)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeyValue

> **⚠️ Important**: KeyValue resources require the JetStream controller to be running in **control-loop mode** (`--control-loop` flag). They are not supported in the default legacy mode.

<sup><sup>[↩ Parent](#jetstreamnatsiov1beta2)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>jetstream.nats.io/v1beta2</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeyValue</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keyvaluespec">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keyvaluestatus">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### KeyValue.spec

<sup><sup>[↩ Parent](#keyvalue)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>account</b></td>
        <td>string</td>
        <td>
          Name of the account to which the Stream belongs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>bucket</b></td>
        <td>string</td>
        <td>
          A unique name for the KV Store.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>compression</b></td>
        <td>boolean</td>
        <td>
          KV Store compression.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>creds</b></td>
        <td>string</td>
        <td>
          NATS user credentials for connecting to servers. Please make sure your controller has mounted the creds on its path.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          The description of the KV Store.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>history</b></td>
        <td>integer</td>
        <td>
          The number of historical values to keep per key.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxBytes</b></td>
        <td>integer</td>
        <td>
          The maximum size of the KV Store in bytes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxValueSize</b></td>
        <td>integer</td>
        <td>
          The maximum size of a value in bytes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keyvaluespecmirror">mirror</a></b></td>
        <td>object</td>
        <td>
          A KV Store mirror.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nkey</b></td>
        <td>string</td>
        <td>
          NATS user NKey for connecting to servers.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keyvaluespecplacement">placement</a></b></td>
        <td>object</td>
        <td>
          The KV Store placement via tags or cluster name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>preventDelete</b></td>
        <td>boolean</td>
        <td>
          When true, the managed KV Store will not be deleted when the resource is deleted.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>preventUpdate</b></td>
        <td>boolean</td>
        <td>
          When true, the managed KV Store will not be updated when the resource is updated.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          The number of replicas to keep for the KV Store in clustered JetStream.<br/>
          <br/>
            <i>Default</i>: 1<br/>
            <i>Minimum</i>: 1<br/>
            <i>Maximum</i>: 5<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keyvaluespecrepublish">republish</a></b></td>
        <td>object</td>
        <td>
          Republish configuration for the KV Store.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>servers</b></td>
        <td>[]string</td>
        <td>
          A list of servers for creating the KV Store.<br/>
          <br/>
            <i>Default</i>: []<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keyvaluespecsourcesindex">sources</a></b></td>
        <td>[]object</td>
        <td>
          A KV Store's sources.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storage</b></td>
        <td>enum</td>
        <td>
          The storage backend to use for the KV Store.<br/>
          <br/>
            <i>Enum</i>: file, memory<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keyvaluespectls">tls</a></b></td>
        <td>object</td>
        <td>
          A client's TLS certs and keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tlsFirst</b></td>
        <td>boolean</td>
        <td>
          When true, the KV Store will initiate TLS before server INFO.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ttl</b></td>
        <td>string</td>
        <td>
          The time expiry for keys.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### KeyValue.spec.mirror

<sup><sup>[↩ Parent](#keyvaluespec)</sup></sup>

A KV Store mirror.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>externalApiPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>externalDeliverPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>filterSubject</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartSeq</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartTime</b></td>
        <td>string</td>
        <td>
          Time format must be RFC3339.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keyvaluespecmirrorsubjecttransformsindex">subjectTransforms</a></b></td>
        <td>[]object</td>
        <td>
          List of subject transforms for this mirror.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### KeyValue.spec.mirror.subjectTransforms[index]

<sup><sup>[↩ Parent](#keyvaluespecmirror)</sup></sup>

A subject transform pair.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>dest</b></td>
        <td>string</td>
        <td>
          Destination subject.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>source</b></td>
        <td>string</td>
        <td>
          Source subject.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### KeyValue.spec.placement

<sup><sup>[↩ Parent](#keyvaluespec)</sup></sup>

The KV Store placement via tags or cluster name.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cluster</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tags</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### KeyValue.spec.republish

<sup><sup>[↩ Parent](#keyvaluespec)</sup></sup>

Republish configuration for the KV Store.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>destination</b></td>
        <td>string</td>
        <td>
          Messages will be additionally published to this subject after Bucket.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>source</b></td>
        <td>string</td>
        <td>
          Messages will be published from this subject to the destination subject.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### KeyValue.spec.sources[index]

<sup><sup>[↩ Parent](#keyvaluespec)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>externalApiPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>externalDeliverPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>filterSubject</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartSeq</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartTime</b></td>
        <td>string</td>
        <td>
          Time format must be RFC3339.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keyvaluespecsourcesindexsubjecttransformsindex">subjectTransforms</a></b></td>
        <td>[]object</td>
        <td>
          List of subject transforms for this mirror.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### KeyValue.spec.sources[index].subjectTransforms[index]

<sup><sup>[↩ Parent](#keyvaluespecsourcesindex)</sup></sup>

A subject transform pair.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>dest</b></td>
        <td>string</td>
        <td>
          Destination subject.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>source</b></td>
        <td>string</td>
        <td>
          Source subject.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### KeyValue.spec.tls

<sup><sup>[↩ Parent](#keyvaluespec)</sup></sup>

A client's TLS certs and keys.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientCert</b></td>
        <td>string</td>
        <td>
          A client's cert filepath. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clientKey</b></td>
        <td>string</td>
        <td>
          A client's key filepath. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>rootCas</b></td>
        <td>[]string</td>
        <td>
          A list of filepaths to CAs. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### KeyValue.status

<sup><sup>[↩ Parent](#keyvalue)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#keyvaluestatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### KeyValue.status.conditions[index]

<sup><sup>[↩ Parent](#keyvaluestatus)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## ObjectStore

> **⚠️ Important**: ObjectStore resources require the JetStream controller to be running in **control-loop mode** (`--control-loop` flag). They are not supported in the default legacy mode.

<sup><sup>[↩ Parent](#jetstreamnatsiov1beta2)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>jetstream.nats.io/v1beta2</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>ObjectStore</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#objectstorespec">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#objectstorestatus">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### ObjectStore.spec

<sup><sup>[↩ Parent](#objectstore)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>account</b></td>
        <td>string</td>
        <td>
          Name of the account to which the Object Store belongs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>bucket</b></td>
        <td>string</td>
        <td>
          A unique name for the Object Store.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>compression</b></td>
        <td>boolean</td>
        <td>
          Object Store compression.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>creds</b></td>
        <td>string</td>
        <td>
          NATS user credentials for connecting to servers. Please make sure your controller has mounted the creds on its path.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          The description of the Object Store.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxBytes</b></td>
        <td>integer</td>
        <td>
          The maximum size of the Store in bytes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>metadata</b></td>
        <td>map[string]string</td>
        <td>
          Additional Object Store metadata.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nkey</b></td>
        <td>string</td>
        <td>
          NATS user NKey for connecting to servers.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#objectstorespecplacement">placement</a></b></td>
        <td>object</td>
        <td>
          The Object Store placement via tags or cluster name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>preventDelete</b></td>
        <td>boolean</td>
        <td>
          When true, the managed Object Store will not be deleted when the resource is deleted.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>preventUpdate</b></td>
        <td>boolean</td>
        <td>
          When true, the managed Object Store will not be updated when the resource is updated.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          The number of replicas to keep for the Object Store in clustered JetStream.<br/>
          <br/>
            <i>Default</i>: 1<br/>
            <i>Minimum</i>: 1<br/>
            <i>Maximum</i>: 5<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>servers</b></td>
        <td>[]string</td>
        <td>
          A list of servers for creating the Object Store.<br/>
          <br/>
            <i>Default</i>: []<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storage</b></td>
        <td>enum</td>
        <td>
          The storage backend to use for the Object Store.<br/>
          <br/>
            <i>Enum</i>: file, memory<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#objectstorespectls">tls</a></b></td>
        <td>object</td>
        <td>
          A client's TLS certs and keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tlsFirst</b></td>
        <td>boolean</td>
        <td>
          When true, the KV Store will initiate TLS before server INFO.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ttl</b></td>
        <td>string</td>
        <td>
          The time expiry for keys.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### ObjectStore.spec.placement

<sup><sup>[↩ Parent](#objectstorespec)</sup></sup>

The Object Store placement via tags or cluster name.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cluster</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tags</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### ObjectStore.spec.tls

<sup><sup>[↩ Parent](#objectstorespec)</sup></sup>

A client's TLS certs and keys.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientCert</b></td>
        <td>string</td>
        <td>
          A client's cert filepath. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clientKey</b></td>
        <td>string</td>
        <td>
          A client's key filepath. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>rootCas</b></td>
        <td>[]string</td>
        <td>
          A list of filepaths to CAs. Should be mounted.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### ObjectStore.status

<sup><sup>[↩ Parent](#objectstore)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#objectstorestatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### ObjectStore.status.conditions[index]

<sup><sup>[↩ Parent](#objectstorestatus)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

# jetstream.nats.io/v1beta1

Resource Types:

- [Stream](#stream)

- [Consumer](#consumer)

- [StreamTemplate](#streamtemplate)

## Stream

<sup><sup>[↩ Parent](#jetstreamnatsiov1beta1)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>jetstream.nats.io/v1beta1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Stream</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#streamspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec

<sup><sup>[↩ Parent](#stream-1)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          The description of the stream.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>discard</b></td>
        <td>enum</td>
        <td>
          When a Stream reach it's limits either old messages are deleted or new ones are denied.<br/>
          <br/>
            <i>Enum</i>: old, new<br/>
            <i>Default</i>: old<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>duplicateWindow</b></td>
        <td>string</td>
        <td>
          The duration window to track duplicate messages for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxAge</b></td>
        <td>string</td>
        <td>
          Maximum age of any message in the stream, expressed in Go's time.Duration format. Empty for unlimited.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxBytes</b></td>
        <td>integer</td>
        <td>
          How big the Stream may be, when the combined stream size exceeds this old messages are removed. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxConsumers</b></td>
        <td>integer</td>
        <td>
          How many Consumers can be defined for a given Stream. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxMsgSize</b></td>
        <td>integer</td>
        <td>
          The largest message that will be accepted by the Stream. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxMsgs</b></td>
        <td>integer</td>
        <td>
          How many messages may be in a Stream, oldest messages will be removed if the Stream exceeds this size. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxMsgsPerSubject</b></td>
        <td>integer</td>
        <td>
          The maximum number of messages per subject.<br/>
          <br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecmirror-1">mirror</a></b></td>
        <td>object</td>
        <td>
          A stream mirror.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          A unique name for the Stream.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>noAck</b></td>
        <td>boolean</td>
        <td>
          Disables acknowledging messages that are received by the Stream.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecplacement-1">placement</a></b></td>
        <td>object</td>
        <td>
          A stream's placement.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          How many replicas to keep for each message.<br/>
          <br/>
            <i>Default</i>: 1<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>retention</b></td>
        <td>enum</td>
        <td>
          How messages are retained in the Stream, once this is exceeded old messages are removed.<br/>
          <br/>
            <i>Enum</i>: limits, interest, workqueue<br/>
            <i>Default</i>: limits<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamspecsourcesindex-1">sources</a></b></td>
        <td>[]object</td>
        <td>
          A stream's sources.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storage</b></td>
        <td>enum</td>
        <td>
          The storage backend to use for the Stream.<br/>
          <br/>
            <i>Enum</i>: file, memory<br/>
            <i>Default</i>: memory<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subjects</b></td>
        <td>[]string</td>
        <td>
          A list of subjects to consume, supports wildcards.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.mirror

<sup><sup>[↩ Parent](#streamspec-1)</sup></sup>

A stream mirror.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>externalApiPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>externalDeliverPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>filterSubject</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartSeq</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartTime</b></td>
        <td>string</td>
        <td>
          Time format must be RFC3339.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.placement

<sup><sup>[↩ Parent](#streamspec-1)</sup></sup>

A stream's placement.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cluster</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tags</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.spec.sources[index]

<sup><sup>[↩ Parent](#streamspec-1)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>externalApiPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>externalDeliverPrefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>filterSubject</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartSeq</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartTime</b></td>
        <td>string</td>
        <td>
          Time format must be RFC3339.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.status

<sup><sup>[↩ Parent](#stream-1)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#streamstatusconditionsindex-1">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Stream.status.conditions[index]

<sup><sup>[↩ Parent](#streamstatus-1)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Consumer

<sup><sup>[↩ Parent](#jetstreamnatsiov1beta1)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>jetstream.nats.io/v1beta1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Consumer</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#consumerspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#consumerstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Consumer.spec

<sup><sup>[↩ Parent](#consumer-1)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>ackPolicy</b></td>
        <td>enum</td>
        <td>
          How messages should be acknowledged.<br/>
          <br/>
            <i>Enum</i>: none, all, explicit<br/>
            <i>Default</i>: none<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ackWait</b></td>
        <td>string</td>
        <td>
          How long to allow messages to remain un-acknowledged before attempting redelivery.<br/>
          <br/>
            <i>Default</i>: 1ns<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>deliverGroup</b></td>
        <td>string</td>
        <td>
          The name of a queue group.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>deliverPolicy</b></td>
        <td>enum</td>
        <td>
            <i>Enum</i>: all, last, new, byStartSequence, byStartTime<br/>
            <i>Default</i>: all<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>deliverSubject</b></td>
        <td>string</td>
        <td>
          The subject to deliver observed messages, when not set, a pull-based Consumer is created.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          The description of the consumer.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>durableName</b></td>
        <td>string</td>
        <td>
          The name of the Consumer.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>filterSubject</b></td>
        <td>string</td>
        <td>
          Select only a specific incoming subjects, supports wildcards.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>flowControl</b></td>
        <td>boolean</td>
        <td>
          Enables flow control.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>heartbeatInterval</b></td>
        <td>string</td>
        <td>
          The interval used to deliver idle heartbeats for push-based consumers, in Go's time.Duration format.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxAckPending</b></td>
        <td>integer</td>
        <td>
          Maximum pending Acks before consumers are paused.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxDeliver</b></td>
        <td>integer</td>
        <td>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartSeq</b></td>
        <td>integer</td>
        <td>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optStartTime</b></td>
        <td>string</td>
        <td>
          Time format must be RFC3339.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>rateLimitBps</b></td>
        <td>integer</td>
        <td>
          Rate at which messages will be delivered to clients, expressed in bit per second.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replayPolicy</b></td>
        <td>enum</td>
        <td>
          How messages are sent.<br/>
          <br/>
            <i>Enum</i>: instant, original<br/>
            <i>Default</i>: instant<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampleFreq</b></td>
        <td>string</td>
        <td>
          What percentage of acknowledgements should be samples for observability.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>streamName</b></td>
        <td>string</td>
        <td>
          The name of the Stream to create the Consumer in.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Consumer.status

<sup><sup>[↩ Parent](#consumer-1)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#consumerstatusconditionsindex-1">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### Consumer.status.conditions[index]

<sup><sup>[↩ Parent](#consumerstatus-1)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## StreamTemplate

<sup><sup>[↩ Parent](#jetstreamnatsiov1beta1)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>jetstream.nats.io/v1beta1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>StreamTemplate</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#streamtemplatespec">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#streamtemplatestatus">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### StreamTemplate.spec

<sup><sup>[↩ Parent](#streamtemplate)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>discard</b></td>
        <td>enum</td>
        <td>
          When a Stream reach it's limits either old messages are deleted or new ones are denied.<br/>
          <br/>
            <i>Enum</i>: old, new<br/>
            <i>Default</i>: old<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>duplicateWindow</b></td>
        <td>string</td>
        <td>
          The duration window to track duplicate messages for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxAge</b></td>
        <td>string</td>
        <td>
          Maximum age of any message in the stream, expressed in Go's time.Duration format. Empty for unlimited.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxBytes</b></td>
        <td>integer</td>
        <td>
          How big the Stream may be, when the combined stream size exceeds this old messages are removed. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxConsumers</b></td>
        <td>integer</td>
        <td>
          How many Consumers can be defined for a given Stream. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxMsgSize</b></td>
        <td>integer</td>
        <td>
          The largest message that will be accepted by the Stream. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxMsgs</b></td>
        <td>integer</td>
        <td>
          How many messages may be in a Stream, oldest messages will be removed if the Stream exceeds this size. -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxStreams</b></td>
        <td>integer</td>
        <td>
          The maximum number of Streams this Template can create, -1 for unlimited.<br/>
          <br/>
            <i>Default</i>: -1<br/>
            <i>Minimum</i>: -1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          A unique name for the Stream Template.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>noAck</b></td>
        <td>boolean</td>
        <td>
          Disables acknowledging messages that are received by the Stream.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          How many replicas to keep for each message.<br/>
          <br/>
            <i>Default</i>: 1<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>retention</b></td>
        <td>enum</td>
        <td>
          How messages are retained in the Stream, once this is exceeded old messages are removed.<br/>
          <br/>
            <i>Enum</i>: limits, interest, workqueue<br/>
            <i>Default</i>: limits<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storage</b></td>
        <td>enum</td>
        <td>
          The storage backend to use for the Stream.<br/>
          <br/>
            <i>Enum</i>: file, memory<br/>
            <i>Default</i>: memory<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subjects</b></td>
        <td>[]string</td>
        <td>
          A list of subjects to consume, supports wildcards.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### StreamTemplate.status

<sup><sup>[↩ Parent](#streamtemplate)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#streamtemplatestatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

### StreamTemplate.status.conditions[index]

<sup><sup>[↩ Parent](#streamtemplatestatus)</sup></sup>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>
