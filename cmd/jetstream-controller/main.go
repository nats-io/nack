// Copyright 2020 The NATS Authors
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
	Version   = "0.2.0"
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
	version := flag.Bool("version", false, "Print the version and exit")
	creds := flag.String("creds", "", "NATS Credentials")
	nkey := flag.String("nkey", "", "NATS NKey")
	server := flag.String("s", "", "NATS Server URL") // required
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s (%s), built %s\n", os.Args[0], Version, GitInfo, BuildTime)
		return nil
	}

	if *server == "" {
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
		KubeIface:       kc,
		JetstreamIface:  jc,
	})

	klog.Infof("Starting %s %s...", os.Args[0], Version)
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
