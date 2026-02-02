# NKey Authentication E2E Test

This kuttl test validates the NKey authentication feature for the NATS Account CRD.

## What This Test Does

1. **Step 00 - Setup**:
   - Deploys a NATS server configured to require NKey authentication
   - Deploys the NACK controller
   - Creates a Kubernetes Secret containing an NKey seed

2. **Step 01 - Account with NKey**:
   - Creates an Account CRD that references the NKey secret
   - Verifies the Account becomes Ready (successfully authenticated)

3. **Step 02 - Stream with NKey Account**:
   - Creates a Stream CRD that uses the NKey-authenticated Account
   - Verifies the Stream is created successfully on the NATS server

4. **Step 03 - Negative Test**:
   - Attempts to create a Stream without NKey authentication
   - Verifies it fails (proving NKey auth is enforced)

## NKey Credentials

The test uses a static NKey pair:
- **Public Key**: `UBRVHFXNO4F7WZUTY73K6FNF7QRPZ6DYJJ2LA6LKSZCSJ7MRM7ZXTMG6`
- **Seed (Private)**: `SUANP3AYWPMX4COR6XL5P2NPNJO4LCJIZ6CGPETRI5IOCHGPE43GNFMZYQ`

⚠️ **Note**: These are test-only credentials and should never be used in production.

## Running This Test

### Locally with kuttl:
```bash
make test-e2e
```

### In CI/CD:
This test runs automatically via the `.github/workflows/e2e.yaml` workflow on every push to main and on pull requests.

## Test Structure

```
tests/nkey-auth/
├── 00-setup.yaml                    # Install NATS with NKey + create Secret
├── 01-account.yaml                  # Test Account with NKey
├── 02-stream.yaml                   # Test Stream using Account
├── 03-noauth-fail.yaml              # Negative test (no auth)
├── nats-nkey.yaml                   # NATS Helm values (NKey config)
├── nkey-secret.yaml                 # K8s Secret with NKey seed
├── nkey-account.yaml                # Account CRD resource
├── asserted-nkey-account.yaml       # Account assertion (Ready)
├── nkey-stream.yaml                 # Stream CRD resource
├── asserted-nkey-stream.yaml        # Stream assertion (Ready)
├── noauth-stream.yaml               # Stream without auth
└── asserted-noauth-stream.yaml      # Stream assertion (Not Ready)
```

## What This Tests

- ✅ NKey seed retrieval from Kubernetes Secrets
- ✅ Account authentication using NKey
- ✅ Stream creation via NKey-authenticated Account
- ✅ Authentication failure without valid NKey
- ✅ Integration with NATS server NKey configuration
- ✅ End-to-end workflow in real Kubernetes cluster
