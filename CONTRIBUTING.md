> [!WARNING]
> This contribution guide is work in progress and is meant to be a location where more developers can contribute.

# Development
The codebase is currently fragmented into the refactored solution using today's standards for creating controllers (when
using the `--control-loop` argument) and the old variant. The old variant is found in `controllers` directory, while the
new code is found in `internal`.

## E2E testing
You may run the entire e2e suite with the accompanying updated image using:
```bash
make test-e2e
```

This command will:
1. Build a local Docker image with your changes
2. Run the full test suite in **legacy controller mode** (using `controllers/` implementation)
3. Run the full test suite in **control-loop mode** (using `internal/controller/` implementation with `--control-loop` flag)

This ensures both controller implementations are tested with your changes.

**Requirements:**
- `kind` must be installed (install via `make install-kind`)
- `kubectl-kuttl` must be installed (install via `kubectl krew install kuttl`)

# CRD Updates

## Generating types
```bash
make generate
```
will update the generated go structs after having updated the types.

## CRD and docs
CRD updates & accompanying documentation is currently updated manually.
TODO to automate this.

