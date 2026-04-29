# Conevent Backend

Leia em Portugues: [README.md](README.md)

Conevent application backend, built with Go, Fiber, PostgreSQL, sqlc, and OpenTelemetry. The API exposes an events CRUD,
OpenAPI/Swagger documentation, Prometheus metrics, and traces exported to Jaeger.

## Features

- Events CRUD at `/events`.
- Health check at `/health`.
- Runtime-generated OpenAPI documentation at `/openapi.json`.
- Swagger UI at `/docs`.
- Prometheus metrics at `/metrics` on the configured metrics port.
- Distributed tracing with OpenTelemetry and OTLP gRPC export to Jaeger.
- Kubernetes manifests with PostgreSQL, Prometheus, Grafana, Loki, Promtail, and Jaeger.

## Architecture

```text
cmd/conevent/              Application entrypoint
config/                    Environment variable loading and validation
internal/api/              Fiber HTTP handlers
internal/service/          Business rules and internal spans
internal/db/               sqlc-generated code, queries, and SQL schema
internal/observability/    Tracing, metrics, and Prometheus server
k8s/                       Kubernetes manifests grouped by domain
insomnia/                  Collection for manual API testing
```

## Requirements

- Go 1.25 or higher.
- Docker and Docker Compose for local containerized execution.
- kubectl and a Kubernetes cluster for cluster execution.
- sqlc to regenerate database code when queries change.

## Configuration

Copy `.env.example` to `.env` and adjust it for your environment.

```env
SERVER_PORT=3000
SERVER_READ_TIMEOUT=5
SERVER_WRITE_TIMEOUT=10
SERVER_IDLE_TIMEOUT=120

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=conevent
DB_SSLMODE=disable

OBS_SERVICE_NAME=conevent-backend
OBS_SERVICE_VERSION=1.0.0
OBS_TRACE_EXPORTER=jaeger
OBS_OTLP_ENDPOINT=localhost:4317
OBS_PROMETHEUS_PORT=9090
```

In Kubernetes, these settings live in `k8s/app.yaml`, and credentials live in `k8s/database.yaml`. The in-cluster OTLP
endpoint is `jaeger:4317`.

## Running Locally

Start a local PostgreSQL instance or use the Docker Compose database. With the database available and `.env` configured:

```bash
go run ./cmd/conevent
```

The API will be available at `http://localhost:3000`.

Useful endpoints:

- `GET http://localhost:3000/health`
- `GET http://localhost:3000/events`
- `POST http://localhost:3000/events`
- `GET http://localhost:3000/openapi.json`
- `GET http://localhost:3000/docs`
- `GET http://localhost:9090/metrics`

## Running With Docker Compose

Compose starts PostgreSQL, Jaeger, and the application:

```bash
docker compose up --build
```

Local access:

- API: `http://localhost:3000`
- Application metrics: `http://localhost:9090/metrics`
- Jaeger UI: `http://localhost:16686`
- PostgreSQL: `localhost:5433`

## Running On Kubernetes

Apply all manifests with Kustomize:

```bash
kubectl apply -k k8s
```

Check the pods and services:

```bash
kubectl get pods
kubectl get svc
```

Services exposed with NodePort:

- Conevent API: `http://<node-ip>:30000`
- Conevent metrics: `http://<node-ip>:30090/metrics`
- Prometheus: `http://<node-ip>:30002`
- Grafana: `http://<node-ip>:30001`
- Jaeger UI: `http://<node-ip>:30086`

Inside the cluster, Prometheus scrapes metrics from `conevent:9090/metrics`. The application sends traces to
`jaeger:4317` using OTLP gRPC.

The manifests were consolidated to reduce the number of files:

- `k8s/app.yaml`: application ConfigMap, API Service, and API Deployment.
- `k8s/database.yaml`: local Secret, PVC, initial schema, PostgreSQL Service, and PostgreSQL Deployment.
- `k8s/observability.yaml`: Prometheus, Grafana, Jaeger, Loki, Promtail, and Grafana PVC.
- `k8s/kustomization.yaml`: entrypoint for applying everything with `kubectl apply -k k8s`.

If the project grows, the natural next step is to split Kustomize into `base/` and `overlays/` for environments such as
`dev`, `staging`, and `prod`.

## Observability

### Traces

The `otelfiber` middleware creates spans for HTTP requests, except `/health`. That route is intentionally ignored to
avoid polluting Jaeger with Kubernetes liveness/readiness probes.

The expected trace path for a request is:

```text
HTTP <method> <route>
└── EventHandler.<Operation>
    └── EventService.<Operation>
        └── sqlc/pgx
```

Handlers receive the context created by `otelfiber`, add an internal span, and propagate the same `context.Context` to
the service layer. The service layer creates child spans such as `EventService.CreateEvent`, `EventService.ListEvents`,
and `EventService.DeleteEvent`, so traces show where the request moved inside the application.

The W3C Trace Context and Baggage propagator is configured globally. When a request arrives with `traceparent`, the
trace is continued; otherwise, a new trace is created.

### Metrics

The application starts a separate HTTP server for metrics on `OBS_PROMETHEUS_PORT`, defaulting to `9090`. Metrics
include:

- Total HTTP requests.
- HTTP request duration.
- Active requests.
- Event operation metrics.
- Application start timestamp.

In Kubernetes, the `conevent` Service exposes port `9090` as `metrics`, and Prometheus is configured to scrape
`conevent:9090`.

### Logs

Application logs are written to stdout/stderr. In Kubernetes, Promtail collects pod logs and sends them to Loki. Grafana
already provisions data sources for Prometheus, Loki, and Jaeger.

In the Promtail `DaemonSet`, the `HOSTNAME` variable is populated from `spec.nodeName`. This ensures Kubernetes
discovery filters by the real node name and finds files under `/var/log/pods`. In kind/containerd clusters, Promtail
also mounts `/var/lib/containerd` and uses the `cri` stage to parse the log format.

To validate Loki ingestion:

```bash
kubectl port-forward svc/loki 3100:3100
curl -G 'http://127.0.0.1:3100/loki/api/v1/query_range' --data-urlencode 'query={app="conevent"}' --data-urlencode 'limit=5'
```

### Grafana

Grafana uses a `PersistentVolumeClaim` named `grafana-pvc`, mounted at `/var/lib/grafana`. This preserves users,
sessions, the internal SQLite database, preferences, and dashboards created through the UI when the pod restarts.

If the cluster is destroyed together with persistent volumes, Grafana data will also be lost. To keep data across full
cluster recreations, use an environment-specific persistent StorageClass or export dashboards as code.

## Database

The main schema is in `internal/db/schema/001_events.sql`. Queries used by sqlc are in `internal/db/queries/event.sql`.

To regenerate generated code:

```bash
sqlc generate
```

## Tests And Validation

Run all tests:

```bash
go test ./...
```

Run tests with the race detector:

```bash
go test -race ./...
```

Validate Go code:

```bash
go vet ./...
```

Validate minimum 95% coverage for application packages:

```bash
sh scripts/coverage.sh
```

The script generates `coverage.out`, filters out only the `cmd/conevent` entrypoint, and keeps application packages,
including sqlc-generated wrappers. `cmd/conevent` is validated by build because it is responsible for starting real
infrastructure and blocking on the HTTP server.

Local build without generating an artifact in the repository:

```bash
go build -o /tmp/conevent-backend ./cmd/conevent
```

Service-layer integration tests use PostgreSQL at `localhost:5433` with database `conevent_test`. If the database is
unavailable, they are skipped automatically; unit tests continue to run.

## Development Flow

1. Adjust SQL schema or queries when necessary.
2. Run `sqlc generate` if SQL files changed.
3. Run `gofmt`, `go test ./...`, and `go vet ./...`.
4. Build the Docker image if publishing to the cluster.
5. Update the image tag in `k8s/app.yaml` when a new image is published.

## Security

The `Secret` in `k8s/database.yaml` contains default base64 credentials only for educational/local use. For any real
environment, replace it with an externally managed Secret and rotate credentials.
