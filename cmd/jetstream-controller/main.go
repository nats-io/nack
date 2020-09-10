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
	"runtime"

	log "github.com/sirupsen/logrus"
	"github.com/nats-io/nack/pkg/jetstream"
)

var (
	showVersion bool
	showHelp    bool
	debug       bool
)

func main() {
	fs := flag.NewFlagSet("jetstream-controller", flag.ExitOnError)
	flag.Usage = func() {
		fmt.Printf("Usage: jetstream-controller [options...]\n\n")
		fs.PrintDefaults()
		fmt.Println()
	}

	opts := &jetstream.Options{}
	fs.BoolVar(&showHelp, "h", false, "Show help")
	fs.BoolVar(&showHelp, "help", false, "Show help")
	fs.BoolVar(&showVersion, "v", false, "Show version")
	fs.BoolVar(&showVersion, "version", false, "Show version")
	fs.BoolVar(&debug, "D", false, "Enable debug mode")
	fs.StringVar(&opts.NatsCredentials, "creds", "", "NATS Credentials")
	fs.StringVar(&opts.NatsServerURL, "s", "nats://localhost:4222", "NATS Server URL")
	fs.StringVar(&opts.ClusterName, "name", "nats", "NATS Cluster Name")
	// fs.StringVar(&opts.ConfigMapName, "cm", "", "NATS Cluster ConfigMap")
	fs.Parse(os.Args[1:])

	switch {
	case showHelp:
		flag.Usage()
		os.Exit(0)
	case showVersion:
		fmt.Printf("NATS JetStream Controller v%s\n", jetstream.Version)
		os.Exit(0)
	}

	// Top level global config.
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)

	controller := jetstream.NewController(opts)
	log.Infof("Starting NATS JetStream Controller v%s", jetstream.Version)
	log.Infof("Go Version: %s", runtime.Version())

	err := controller.Run(context.Background())
	if err != nil && err != context.Canceled {
		log.Errorf(err.Error())
		os.Exit(1)
	}
}
