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
	"os"
	"os/signal"
	"syscall"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nats-io/nack/controllers/jetstream"
	clientset "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	klog "k8s.io/klog/v2"
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
	kubeConfig := flag.String("kubeconfig", "", "Path to kubeconfig")
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
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s (%s), built %s\n", os.Args[0], Version, GitInfo, BuildTime)
		return nil
	}

	if *server == "" && !*crdConnect {
		return errors.New("NATS Server URL is required")
	}

	var config *rest.Config
	var err error
	if *kubeConfig == "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeConfig)
		if err != nil {
			return err
		}
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
