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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nats-io/nack/controllers/jetstream"
	clientset "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned"
	informers "github.com/nats-io/nack/pkg/jetstream/generated/informers/externalversions"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	BuildTime = "not-set"
)

var (
	showVersion bool
	showHelp    bool
	debug       bool
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	fs := flag.NewFlagSet("jetstream-controller", flag.ExitOnError)
	flag.Usage = func() {
		fmt.Printf("Usage: jetstream-controller [options...]\n\n")
		fs.PrintDefaults()
		fmt.Println()
	}

	fs.BoolVar(&showHelp, "h", false, "Show help")
	fs.BoolVar(&showHelp, "help", false, "Show help")
	fs.BoolVar(&showVersion, "v", false, "Show version")
	fs.BoolVar(&showVersion, "version", false, "Show version")
	fs.BoolVar(&debug, "D", false, "Enable debug mode")
	//fs.StringVar(&opts.NatsCredentials, "creds", "", "NATS Credentials")
	//fs.StringVar(&opts.NatsServerURL, "s", "nats://localhost:4222", "NATS Server URL")
	//fs.StringVar(&opts.ClusterName, "name", "nats", "NATS Cluster Name")
	//fs.StringVar(&opts.ConfigMapName, "cm", "", "NATS Cluster ConfigMap")
	fs.Parse(os.Args[1:])

	switch {
	case showHelp:
		flag.Usage()
		os.Exit(0)
	case showVersion:
		fmt.Printf("NATS JetStream Controller v1")
		os.Exit(0)
	}

	var err error
	var config *rest.Config
	if kubeconfig := os.Getenv("KUBERNETES_CONFIG_FILE"); kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return err
	}

	// Top level global config.
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)

	log.Infof("Starting NATS JetStream Controller v1")
	log.Infof("Go Version: %s", runtime.Version())
	log.Infof("BuildTime: %s", BuildTime)

	kcs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	cs, err := clientset.NewForConfig(config)
	if err != nil {
		return err
	}

	aecs, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctrl, err := jetstream.NewController(jetstream.Options{
		Ctx:            ctx,
		KubeIface:      kcs,
		APIExtIface:    aecs,
		JetstreamIface: cs.JetstreamV1(),

		InformerFactory: informers.NewSharedInformerFactory(cs, 30*time.Second),
	})
	if err != nil {
		return err
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
