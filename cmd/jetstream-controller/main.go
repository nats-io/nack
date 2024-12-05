// Copyright 2020-2023 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/nats-io/nack/controllers/jetstream"
	"github.com/nats-io/nack/internal/controller"
	jetstreamnatsiov1beta2 "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	clientset "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	klog "k8s.io/klog/v2"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"syscall"
	"time"
)

var (
	BuildTime = "build-time-not-set"
	GitInfo   = "gitinfo-not-set"
	Version   = "not-set"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	klog.InitFlags(nil)

	// Explicitly register controller-runtime flags
	ctrl.RegisterFlags(nil)

	namespace := flag.String("namespace", v1.NamespaceAll, "Restrict to a namespace")
	version := flag.Bool("version", false, "Print the version and exit")
	creds := flag.String("creds", "", "NATS Credentials")
	nkey := flag.String("nkey", "", "NATS NKey")
	cert := flag.String("tlscert", "", "NATS TLS public certificate")
	key := flag.String("tlskey", "", "NATS TLS private key")
	ca := flag.String("tlsca", "", "NATS TLS certificate authority chain")
	tlsfirst := flag.Bool("tlsfirst", false, "If enabled, forces explicit TLS without waiting for Server INFO")
	server := flag.String("s", "", "NATS Server URL")
	crdConnect := flag.Bool("crd-connect", false, "If true, then NATS connections will be made from CRD config, not global config")
	cleanupPeriod := flag.Duration("cleanup-period", 30*time.Second, "Period to run object cleanup")
	readOnly := flag.Bool("read-only", false, "Starts the controller without causing changes to the NATS resources")
	controlLoop := flag.Bool("control-loop", false, "Experimental: Run controller with a full reconciliation control loop.")

	flag.Parse()

	if *version {
		fmt.Printf("%s version %s (%s), built %s\n", os.Args[0], Version, GitInfo, BuildTime)
		return nil
	}

	if *server == "" && !*crdConnect {
		return errors.New("NATS Server URL is required")
	}

	config, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("get kubernetes rest config: %w", err)
	}

	if *controlLoop {
		klog.Warning("Starting jetStream controller in experimental control loop mode")
		natsCfg := &controller.NatsConfig{
			CRDConnect:  *crdConnect,
			ClientName:  "jetstream-controller",
			Credentials: *creds,
			NKey:        *nkey,
			ServerURL:   *server,
			CAs:         []string{*ca},
			Certificate: *cert,
			Key:         *key,
			TLSFirst:    *tlsfirst,
		}

		controllerCfg := &controller.Config{
			ReadOnly:  *readOnly,
			Namespace: *namespace,
		}
		return runControlLoop(config, natsCfg, controllerCfg)
	}

	// K8S API Client.
	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// JetStream CRDs client.
	jc, err := clientset.NewForConfig(config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl := jetstream.NewController(jetstream.Options{
		// FIXME: Move context to be param from Run
		// to avoid keeping state in options.
		Ctx:             ctx,
		NATSCredentials: *creds,
		NATSNKey:        *nkey,
		NATSServerURL:   *server,
		NATSCA:          *ca,
		NATSCertificate: *cert,
		NATSKey:         *key,
		NATSTLSFirst:    *tlsfirst,
		KubeIface:       kc,
		JetstreamIface:  jc,
		Namespace:       *namespace,
		CRDConnect:      *crdConnect,
		CleanupPeriod:   *cleanupPeriod,
		ReadOnly:        *readOnly,
	})

	klog.Infof("Starting %s v%s...", os.Args[0], Version)
	if *readOnly {
		klog.Infof("Running in read-only mode: JetStream state in server will not be changed")
	}
	go handleSignals(cancel)
	return ctrl.Run()
}

func runControlLoop(config *rest.Config, natsCfg *controller.NatsConfig, controllerCfg *controller.Config) error {

	// Setup scheme
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(jetstreamnatsiov1beta2.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
		Logger: klog.NewKlogr().WithName("controller-runtime"),
		// TODO Add full configuration
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	err = controller.RegisterAll(mgr, natsCfg, controllerCfg)
	if err != nil {
		return fmt.Errorf("register jetstream controllers: %w", err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	klog.Info("starting manager")
	return mgr.Start(ctrl.SetupSignalHandler())
}

func handleSignals(cancel context.CancelFunc) {
	sigc := make(chan os.Signal, 2)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	for sig := range sigc {
		switch sig {
		case syscall.SIGINT:
			os.Exit(130)
		case syscall.SIGTERM:
			cancel()
			return
		}
	}
}
