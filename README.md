# Conevent Backend

Read in English: [README.en.md](README.en.md)

Backend da aplicação Conevent, desenvolvido em Go com Fiber, PostgreSQL, sqlc e OpenTelemetry. A API expõe um CRUD de eventos, documentação OpenAPI/Swagger, métricas Prometheus e traces enviados para Jaeger.

## Funcionalidades

- CRUD de eventos em `/events`.
- Health check em `/health`.
- Documentação OpenAPI gerada em tempo de execução em `/openapi.json`.
- Swagger UI em `/docs`.
- Métricas Prometheus em `/metrics` na porta de métricas configurada.
- Tracing distribuído com OpenTelemetry e exportação OTLP gRPC para Jaeger.
- Manifests Kubernetes com PostgreSQL, Prometheus, Grafana, Loki, Promtail e Jaeger.

## Arquitetura

```text
cmd/conevent/              Entrada da aplicação
config/                    Carregamento e validação das variáveis de ambiente
internal/api/              Handlers HTTP Fiber
internal/service/          Regras de negócio e spans internos
internal/db/               Código gerado pelo sqlc, queries e schema SQL
internal/observability/    Tracing, métricas e servidor Prometheus
k8s/                       Manifests Kubernetes consolidados por domínio
insomnia/                  Coleção para testes manuais da API
```

## Requisitos

- Go 1.25 ou superior.
- Docker e Docker Compose para execução local containerizada.
- kubectl e um cluster Kubernetes para execução em cluster.
- sqlc para regenerar código de banco quando as queries mudarem.

## Configuração

Copie `.env.example` para `.env` e ajuste conforme o ambiente.

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

No Kubernetes, essas configurações ficam em `k8s/app.yaml` e as credenciais ficam em `k8s/database.yaml`. O endpoint OTLP do cluster é `jaeger:4317`.

## Rodando Localmente

Suba um PostgreSQL local ou use o banco do Docker Compose. Com o banco disponível e `.env` configurado:

```bash
go run ./cmd/conevent
```

A API ficará disponível em `http://localhost:3000`.

Endpoints úteis:

- `GET http://localhost:3000/health`
- `GET http://localhost:3000/events`
- `POST http://localhost:3000/events`
- `GET http://localhost:3000/openapi.json`
- `GET http://localhost:3000/docs`
- `GET http://localhost:9090/metrics`

## Rodando com Docker Compose

O Compose sobe PostgreSQL, Jaeger e a aplicação:

```bash
docker compose up --build
```

Acessos locais:

- API: `http://localhost:3000`
- Métricas da aplicação: `http://localhost:9090/metrics`
- Jaeger UI: `http://localhost:16686`
- PostgreSQL: `localhost:5433`

## Rodando no Kubernetes

Aplique todos os manifests com Kustomize:

```bash
kubectl apply -k k8s
```

Verifique os pods:

```bash
kubectl get pods
kubectl get svc
```

Serviços expostos por NodePort:

- API Conevent: `http://<node-ip>:30000`
- Métricas Conevent: `http://<node-ip>:30090/metrics`
- Prometheus: `http://<node-ip>:30002`
- Grafana: `http://<node-ip>:30001`
- Jaeger UI: `http://<node-ip>:30086`

Dentro do cluster, o Prometheus coleta métricas em `conevent:9090/metrics`. A aplicação envia traces para `jaeger:4317` usando OTLP gRPC.

Os manifests foram consolidados para reduzir a quantidade de arquivos:

- `k8s/app.yaml`: ConfigMap da aplicação, Service e Deployment da API.
- `k8s/database.yaml`: Secret local, PVC, schema inicial, Service e Deployment do PostgreSQL.
- `k8s/observability.yaml`: Prometheus, Grafana, Jaeger, Loki, Promtail e PVC do Grafana.
- `k8s/kustomization.yaml`: ponto de entrada para aplicar tudo com `kubectl apply -k k8s`.

Se o projeto crescer, o próximo passo natural é separar `base/` e `overlays/` no Kustomize para ambientes como `dev`, `homolog` e `prod`.

## Observabilidade

### Traces

O middleware `otelfiber` cria spans para as requisições HTTP, exceto `/health`. Essa rota é ignorada de propósito para não poluir o Jaeger com probes de liveness/readiness do Kubernetes.

O caminho esperado de uma requisição rastreada é:

```text
HTTP <método> <rota>
└── EventHandler.<Operação>
    └── EventService.<Operação>
        └── sqlc/pgx
```

Os handlers recebem o contexto criado pelo `otelfiber`, adicionam um span interno e propagam esse mesmo `context.Context` para a camada de serviço. A camada de serviço cria spans filhos como `EventService.CreateEvent`, `EventService.ListEvents` e `EventService.DeleteEvent`. Assim, o trace mostra por onde a requisição passou dentro da aplicação.

O propagador W3C Trace Context e Baggage é configurado globalmente. Quando uma requisição chega com `traceparent`, o trace é continuado; quando não chega, um novo trace é criado.

### Métricas

A aplicação abre um servidor HTTP separado para métricas na porta `OBS_PROMETHEUS_PORT`, por padrão `9090`. As métricas incluem:

- Total de requisições HTTP.
- Duração de requisições HTTP.
- Requisições ativas.
- Métricas de operações de eventos.
- Timestamp de início da aplicação.

No Kubernetes, o Service `conevent` expõe a porta `9090` como `metrics`, e o Prometheus está configurado para coletar `conevent:9090`.

### Logs

Os logs da aplicação são escritos em stdout/stderr. No Kubernetes, o Promtail coleta logs dos pods e envia para Loki. O Grafana já provisiona fontes de dados para Prometheus, Loki e Jaeger.

No `DaemonSet` do Promtail, a variável `HOSTNAME` é preenchida com `spec.nodeName`. Isso garante que a descoberta Kubernetes filtre pelo nome real do node e encontre os arquivos em `/var/log/pods`. Em clusters kind/containerd, o Promtail também monta `/var/lib/containerd` e usa o estágio `cri` para interpretar o formato dos logs.

Para validar a ingestão no Loki:

```bash
kubectl port-forward svc/loki 3100:3100
curl -G 'http://127.0.0.1:3100/loki/api/v1/query_range' --data-urlencode 'query={app="conevent"}' --data-urlencode 'limit=5'
```

### Grafana

O Grafana usa um `PersistentVolumeClaim` chamado `grafana-pvc`, montado em `/var/lib/grafana`. Isso preserva usuários, sessões, banco SQLite interno, preferências e dashboards criados pela interface quando o pod reinicia.

Se o cluster for destruído junto com os volumes persistentes, os dados do Grafana também serão perdidos. Para manter os dados entre recriações completas do cluster, use uma StorageClass persistente do ambiente ou exporte os dashboards como código.

## Banco de Dados

O schema principal está em `internal/db/schema/001_events.sql`. As queries usadas pelo sqlc ficam em `internal/db/queries/event.sql`.

Para regenerar o código gerado:

```bash
sqlc generate
```

## Testes e Validação

Rodar todos os testes:

```bash
go test ./...
```

Rodar testes com detector de corrida:

```bash
go test -race ./...
```

Validar código Go:

```bash
go vet ./...
```

Validar cobertura mínima de 95% nos pacotes da aplicação:

```bash
sh scripts/coverage.sh
```

O script gera `coverage.out`, filtra apenas o entrypoint `cmd/conevent` e mantém os pacotes de aplicação, incluindo os wrappers gerados pelo sqlc. O `cmd/conevent` é validado por build porque é responsável por subir infraestrutura real e bloquear no servidor HTTP.

Build local sem gerar artefato no repositório:

```bash
go build -o /tmp/conevent-backend ./cmd/conevent
```

Os testes de integração da camada de serviço usam PostgreSQL em `localhost:5433` com banco `conevent_test`. Se o banco não estiver disponível, eles são ignorados automaticamente; os testes unitários continuam executando.

## Fluxo de Desenvolvimento

1. Ajuste schema ou queries SQL quando necessário.
2. Execute `sqlc generate` se arquivos SQL forem alterados.
3. Rode `gofmt`, `go test ./...` e `go vet ./...`.
4. Gere a imagem Docker se for publicar no cluster.
5. Atualize a tag da imagem em `k8s/app.yaml` quando uma nova imagem for publicada.

## Segurança

O `Secret` em `k8s/database.yaml` contém credenciais padrão em base64 apenas para ambiente educacional/local. Para qualquer ambiente real, substitua por um Secret gerenciado fora do repositório e rotacione as credenciais.
