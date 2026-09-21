# Policy Snapshot

Policy Snapshot is a runnable, polyglot service composed of a caller-facing gateway, a policy
decision service, and a versioned policy registry. The decision service evaluates requests against
bundled policy revision 1. The registry exposes synthetic revision 2 bundles for local development
and integration testing.

## Architecture

The default deployment runs the gateway and decision service. The optional `refresh` Compose
profile also starts the independently deployed registry.

```text
External caller -> decision-gateway -> decision-service -> bundled snapshot (revision 1)
Policy registry -> versioned bundle API (revision 2 fixture)
```

The gateway forwards the public API without parsing policy payloads. The decision service uses its
bundled policy data. The registry is an optional fixture service and is not part of the default
serving deployment.

See [docs/architecture.md](docs/architecture.md),
[docs/bundle-protocol.md](docs/bundle-protocol.md), and
[docs/fixtures.md](docs/fixtures.md) for repository reference material.

## Quick start

Start the gateway and decision service with:

```bash
make run
```

Start all three services, including the registry, with:

```bash
make run-refresh
```

Automatic policy refresh is disabled in the checked-in configuration, so decision responses use
policy revision 1 in both deployments.

Then query the decision API:

```bash
curl -s http://localhost:8080/v1/decisions \
  -H 'content-type: application/json' \
  -d '{"subject_id":"subject-123","operation":"export","region":"standard"}'
```

Run containerized unit tests and black-box integration tests with:

```bash
make test
make integration-test
```

## Services

| Service | Port | Responsibility |
| --- | ---: | --- |
| `decision-gateway` | 8080 | Transparent caller-facing proxy |
| `decision-service` | 8081 | Policy evaluation service |
| `policy-registry` | 8082 | Versioned bundle fixture source |

All services expose `/health/live` and `/health/ready`. The public API is:

```text
POST /v1/decisions
```
