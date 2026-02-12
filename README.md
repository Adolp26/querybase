# QueryBase API

API em Go que transforma queries SQL cadastradas em endpoints REST com cache Redis e suporte a multiplos bancos de dados.

## Arquitetura

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Sistemas     │────▶│    API (Go)     │────▶│  Oracle / MySQL │
│    Clientes     │     │   porta 8080    │     │   PostgreSQL    │
└─────────────────┘     └────────┬────────┘     └─────────────────┘
                                 │
                ┌────────────────┼────────────────┐
                ▼                ▼                ▼
          ┌──────────┐    ┌──────────┐    ┌──────────────┐
          │  Redis   │    │ Postgres │    │ QueryBase Web│
          │ (Cache)  │    │(Metadata)│    │   (Laravel)  │
          └──────────┘    └──────────┘    └──────────────┘
```

## Stack

- **Go 1.21+** com Gin
- **Redis** para cache de resultados
- **PostgreSQL** para metadados (queries, datasources, logs)
- **Datasources**: Oracle, PostgreSQL e MySQL
- **Criptografia**: AES-256-GCM para senhas de datasources

## Como rodar

```bash
# Copiar e editar configuracoes
cp configs/config.example.yaml configs/config.yaml

# Definir chave de criptografia (mesma usada no Laravel)
export QUERYBASE_ENCRYPTION_KEY="sua-chave-base64-de-32-bytes"

# Rodar
go run ./cmd/api/main.go
```

## Endpoints

| Metodo | Endpoint | Descricao |
|--------|----------|-----------|
| GET | `/health` | Health check |
| POST | `/api/test-connection` | Testa conexao com datasource |
| GET | `/api/queries` | Lista queries disponiveis |
| GET | `/api/query/:slug` | Executa query por slug |

## Exemplo de uso

```bash
# Listar queries
curl http://localhost:8080/api/queries

# Executar query sem parametros
curl http://localhost:8080/api/query/vendas-total

# Executar query com parametros
curl "http://localhost:8080/api/query/vendas-por-periodo?data_inicio=2024-01-01&data_fim=2024-12-31"
```

### Resposta

```json
{
  "data": [
    {"employee_id": 1, "first_name": "John", "last_name": "Doe"}
  ],
  "meta": {
    "slug": "employees-all",
    "name": "Listar Funcionarios",
    "datasource": "oracle-producao",
    "driver": "oracle",
    "count": 100,
    "cache_hit": true,
    "duration": "2.5ms",
    "parameters": {}
  }
}
```

## Estrutura do projeto

```
querybase-api/
├── cmd/api/main.go              # Entrada da aplicacao
├── configs/
│   ├── config.example.yaml      # Configuracoes exemplo
│   └── config.yaml              # Configuracoes locais (gitignore)
├── internal/
│   ├── crypto/                  # Criptografia AES-256-GCM
│   ├── database/
│   │   ├── connection_manager.go  # Conexoes dinamicas (Oracle, MySQL, Postgres)
│   │   ├── postgres.go            # Cliente PostgreSQL (metadados)
│   │   └── redis.go               # Cliente Redis (cache)
│   ├── handlers/
│   │   ├── connection.go          # Teste de conexao
│   │   ├── dynamic_query.go       # Execucao de queries
│   │   └── health.go              # Health check
│   ├── middleware/                # Auth, CORS, Rate Limit, Security
│   ├── models/                    # Structs de dados
│   ├── repository/                # Acesso ao PostgreSQL
│   └── services/                  # Cache service
└── pkg/config/                    # Loader de configuracao
```

## Cache

- TTL configuravel por query (campo `cache_ttl` no banco)
- Fallback para TTL global do Redis quando nao definido
- Cache key deterministica: `query:{slug}:{param1}={valor1}:{param2}={valor2}`
- Status reportado no campo `meta.cache_hit` da resposta

## Criptografia

Senhas de datasources sao armazenadas encriptadas no PostgreSQL com AES-256-GCM. A mesma chave (`QUERYBASE_ENCRYPTION_KEY`) deve ser configurada no Laravel e na API Go.

Se a chave nao estiver definida, a API continua funcionando (senhas sao usadas como estao no banco).

## Seguranca

| Recurso | Config |
|---------|--------|
| API Key Auth | `security.enable_auth` + `security.api_keys` |
| Rate Limiting | `security.enable_rate_limit` + `security.requests_per_minute` |
| CORS | `security.allowed_origins` |
| Input Sanitization | Automatico (bloqueia SQL injection, XSS) |
| Security Headers | Automatico (nosniff, DENY, XSS-Protection) |

## Tipos de parametros

| Tipo | Formato | Exemplo |
|------|---------|---------|
| `string` | Texto livre | `nome=Joao` |
| `integer` | Numero inteiro | `id=123` |
| `number` | Decimal | `valor=99.90` |
| `date` | YYYY-MM-DD | `data=2024-01-15` |
| `datetime` | YYYY-MM-DD HH:MM:SS | `ts=2024-01-15 10:30:00` |
| `boolean` | true/false | `ativo=true` |

## Banco de metadados (PostgreSQL)

| Tabela | Descricao |
|--------|-----------|
| `datasources` | Fontes de dados cadastradas |
| `queries` | Queries SQL com slug, TTL, timeout |
| `query_parameters` | Parametros tipados por query |
| `query_executions` | Log de execucoes (duracao, cache hit, erros) |

## Build

```bash
go build -o querybase ./cmd/api/main.go
```
