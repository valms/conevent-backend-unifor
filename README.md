# Conevent Backend

Este é o backend da aplicação Conevent, desenvolvido em Go utilizando o framework Fiber.

## Estrutura do Projeto

Seguimos o layout padrão de projetos Go:

```
/
├── cmd/
│   └── conevent/          # Ponto de entrada da aplicação (main.go)
├── internal/              # Lógica privada da aplicação e de negócios
│   ├── api/               # Handlers e rotas do Fiber
│   ├── db/                # Código gerado pelo sqlc e migrações
│   │   ├── queries/       # Arquivos .sql para o sqlc
│   │   └── schema/        # Definições do esquema do banco de dados
│   └── service/           # Lógica de negócios / Casos de uso
├── pkg/                   # Código de biblioteca pública (seguro para importação por outros projetos)
├── config/                # Carregamento de configuração e estruturas
├── sqlc.yaml              # Configuração do sqlc
├── go.mod / go.sum        # Gerenciamento de dependências
└── README.md              # Documentação geral do projeto (PT-BR)
```

## Comandos de Desenvolvimento

```bash
# Executar a aplicação localmente
go run ./cmd/conevent

# Executar testes com detector de corrida
go test -race ./...

# Gerar código Go a partir das consultas SQL usando sqlc
sqlc generate

# Executar linters (garantir que o golangci-lint está instalado)
golangci-lint run

# Atualizar dependências
go mod tidy
```

### Política de Linguagem (CRÍTICA)

* **Código Fonte:** Todo o código (variáveis, funções, tipos, logs e comentários de código) **DEVE** ser escrito em *
  *Inglês**.
* **Documentação:** O `README.md` e a documentação de alto nível do projeto **DEVEM** ser escritos em **Português (
  PT-BR)**.

## Licença

Este projeto está licenciado sob os termos da licença MIT.