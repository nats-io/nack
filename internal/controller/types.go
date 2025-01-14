package controller

const (
	readyCondType        = "Ready"
	accountFinalizer     = "account.nats.io/finalizer"
	streamFinalizer      = "stream.nats.io/finalizer"
	keyValueFinalizer    = "kv.nats.io/finalizer"
	objectStoreFinalizer = "objectstore.nats.io/finalizer"
	consumerFinalizer    = "consumer.nats.io/finalizer"

	stateAnnotationConsumer = "consumer.nats.io/state"
	stateAnnotationKV       = "kv.nats.io/state"
	stateAnnotationOS       = "objectstore.nats.io/state"
	stateAnnotationStream   = "stream.nats.io/state"

	stateReady       = "Ready"
	stateReconciling = "Reconciling"
	stateErrored     = "Errored"
	stateFinalizing  = "Finalizing"
)
