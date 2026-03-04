# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

NACK (NATS Controllers for Kubernetes) is a Go-based Kubernetes operator that manages NATS JetStream resources (Streams, Consumers, Accounts, KeyValue, ObjectStore) via CRDs. It also includes a NATS server config reloader sidecar and a boot config init container.

## Build & Test Commands

```bash
make build                          # Build all binaries
make jetstream-controller           # Build main controller (with race detector)
make nats-server-config-reloader    # Build config reloader sidecar
make nats-boot-config               # Build boot config utility
make test                           # Run unit tests (go vet + envtest + go test)
make test-e2e                       # Run E2E tests with kuttl (requires kind)
make generate                       # Regenerate K8s clientset and deepcopy code
make clean                          # Remove built binaries
```

Run a single test package:
```bash
go test -race -cover -count=1 -timeout 30s ./internal/controller/...
go test -race -cover -count=1 -timeout 30s ./controllers/jetstream/...
```

Run a single test:
```bash
go test -race -count=1 -timeout 30s -run TestMyFunction ./internal/controller/...
```

Format code: `go fmt ./...`

## Architecture

### Two Controller Modes

The `jetstream-controller` binary runs in one of two modes:

- **Legacy mode** (default): Event-driven queue processing using custom informer factories. Supports only Stream and Consumer. Code in `controllers/jetstream/`.
- **Control-loop mode** (`--control-loop`): Full controller-runtime reconciliation loop. Supports all resource types (Stream, Consumer, Account, KeyValue, ObjectStore). Code in `internal/controller/`.

Entry point: `cmd/jetstream-controller/main.go` — the `--control-loop` flag selects which mode to run.

### CRD Types

All CRDs are API version `jetstream.nats.io/v1beta2`, defined in `pkg/jetstream/apis/jetstream/v1beta2/`:
- `streamtypes.go`, `consumertypes.go`, `accounttypes.go`, `keyvaluetypes.go`, `objectstoretypes.go`

Generated client code lives in `pkg/jetstream/generated/` — do not edit manually, run `make generate`.

### Controller Patterns

Controllers follow standard Kubernetes operator patterns:
- **Finalizers** for safe deletion cleanup (defined in `internal/controller/types.go`)
- **Status conditions** (Ready/Errored) tracked on each resource
- **State annotations** for reconciliation state tracking (Ready, Reconciling, Errored, Finalizing)
- **Idempotent reconciliation** — operations must be safe to retry
- **Owner references** for parent-child relationships (e.g., Consumer → Stream)

### Other Components

- `pkg/natsreloader/` — watches config files and sends SIGHUP to reload NATS server
- `pkg/bootconfig/` — init container for node-level network config

## Key Dependencies

- `sigs.k8s.io/controller-runtime` — Kubernetes controller framework (control-loop mode)
- `k8s.io/client-go` — Kubernetes client (legacy mode)
- `github.com/nats-io/nats.go` — NATS client
- `github.com/nats-io/jsm.go` — JetStream management

## Review Focus Areas

When reviewing changes, pay attention to:
- Kubernetes controller reconciliation correctness (idempotency, status updates, error handling)
- Proper use of controller-runtime patterns (watches, predicates, ownership references)
- Go error handling (wrapped errors, sentinel errors, no swallowed errors)

## Local Development

```bash
# Build and run against a local kubeconfig
make jetstream-controller
./jetstream-controller -kubeconfig ~/.kube/config -s nats://localhost:4222

# Start a local JetStream-enabled NATS server
nats-server -DV -js

# Increase log verbosity (klog flags)
./jetstream-controller -kubeconfig ~/.kube/config -s nats://localhost:4222 -v=10
```
