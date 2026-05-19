# AGENTS.md — Rosetta

## Run commands

```bash
# Start infra (MySQL)
docker compose up -d mysql

# Run backend (must be in backend/ dir — static files use relative paths)
cd backend && go run .

# Test single endpoint (get token first)
TOKEN=$(curl -s http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/models

# Build binary (gitignored)
cd backend && go build -o rosetta-server .

# Quality check
cd backend && go vet ./...
```

## Architecture

- **Module**: `rosetta` (not github.com/...). All Go code under `backend/`.
- **Entrypoint**: `backend/main.go` — wires services, handlers, middleware, static files.
- **Config**: `backend/config.yaml` — Viper reads from cwd/env. Override with `ROSETTA_*` env vars.
- **DB**: GORM AutoMigrate on startup. MySQL via Docker, or SQLite (`type: sqlite`) for zero-dependency dev.
- **Seeder**: On first run, creates 4 roles + admin user (`admin`/`admin123`). Idempotent (checks existence).
- **Frontend**: Vanilla JS SPA served from `../frontend/` by the Go server. `NoRoute` handler serves `index.html` for SPA routing. No build step.
- **API response**: `{code: 0, message: "ok", data: ...}`. Non-zero code = error.
- **DDL renderers**: Strategy pattern in `ddl/`. `MySQLRenderer` and `GaussDBRenderer`.

## Gotchas

### `LogicalModel.TabName` field naming

The table name field is `TabName` (Go struct) with GORM column `table_name`.  
**Never use `.TableName`** — that calls the GORM method which returns `"logical_model"` (the GORM table name), not the actual table name.  
Always use `.TabName` in Go code. JSON output is `table_name` regardless.

### SQLite driver: pure Go, no CGO

Uses `github.com/glebarez/sqlite` (wraps `modernc.org/sqlite`) because standard `go-sqlite3` requires CGO and gcc. The driver works identically — same GORM API.

### RBAC many2many column names

`RbacUser.Roles` uses explicit join tags:
```go
gorm:"many2many:rbac_user_role;joinForeignKey:UserID;joinReferences:RoleID"
```
Without these, GORM defaults to `rbac_user_id`/`rbac_role_id` which don't match the actual columns `user_id`/`role_id`.

### Boolean GORM zero-value trap

`ModelColumn.Nullable` and `IsPrimaryKey` are `bool` with `default:false`.  
When creating via `tx.Create(&mc)`, `false` is properly stored (unlike the original `default:true` which caused every field to appear nullable).  
For new models with boolean fields, always use `default:false` or `*bool` if the default should truly be `true`.

### Docker registry mirror

`docker-compose.yml` uses `docker.m.daocloud.io/library/mysql:8.0` — a Chinese mirror.  
If pulling fails, try `mysql:8.0` directly from Docker Hub.

### Frontend entry quirks

- Must run `go run .` from `backend/` directory — static file paths are relative (`../frontend/`).
- No npm/node needed — pure HTML/CSS/JS.
- Hash-based routing: `#/login`, `#/users`, `#/instances`, `#/dicts`, `#/models`, `#/models/:id`, `#/help`.
- Page functions return HTML strings in `js/pages/*.js`. Functions are registered globally (no module system).

### API authentication

All endpoints except `POST /api/auth/login` require `Authorization: Bearer <token>`.  
Middleware chain: CORS → AuditLog → [route-specific AuthRequired].

### Config file has dev credentials

`config.yaml` contains plaintext MySQL password (`rosetta123`) and JWT secret.  
For production, override via environment variables: `ROSETTA_DB_PASSWORD`, `ROSETTA_JWT_SECRET`.
