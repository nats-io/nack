package controller

import (
	"fmt"
	ctrl "sigs.k8s.io/controller-runtime"
)

// The Config contains parameters to be considered by the registered controllers.
//
// ReadOnly prevents controllers from actually applying changes NATS resources.
//
// Namespace restricts the controller to resources of the given namespace.
type Config struct {
	ReadOnly  bool
	Namespace string
}

// RegisterAll registers all available jetStream controllers to the manager.
// natsCfg is specific to the nats server connection.
// controllerCfg defines behaviour of the registered controllers.
func RegisterAll(mgr ctrl.Manager, clientConfig *NatsConfig, config *Config) error {

	// Controllers need access to a nats client in pedantic mode
	js, err := CreateJetStreamClient(clientConfig, true)
	if err != nil {
		return fmt.Errorf("create jetstream client: %w", err)
	}

	// Register controllers
	if err := (&AccountReconciler{
		Client:    mgr.GetClient(),
		Config:    config,
		Scheme:    mgr.GetScheme(),
		JetStream: js,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create account controller: %w", err)
	}

	if err := (&ConsumerReconciler{
		Client:    mgr.GetClient(),
		Config:    config,
		Scheme:    mgr.GetScheme(),
		JetStream: js,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create consumer controller: %w", err)
	}

	if err := (&StreamReconciler{
		Client:    mgr.GetClient(),
		Config:    config,
		Scheme:    mgr.GetScheme(),
		JetStream: js,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create stream controller: %w", err)
	}

	return nil
}
