# OneLab-API

OneLab-API is a small Go API gateway that brings self-hosted services behind a single authenticated HTTP interface. It currently supports:

- auth integrations for token issuance and validation
- file integrations for Nextcloud and Paperless-ngx
- a local shell backend for executing Docker commands through the host Docker CLI

The project is designed to run as a single lightweight service and can be deployed with Docker, Docker Compose, or a self-hosted GitHub Actions runner.

---

## Features

- JWT-based auth via the `jwt` integration
- OAuth/OIDC client-credential auth via the `oauth` integration
- file lookup and transfer between configured services
- local Docker command execution via `dockerCmd`
- configuration-driven enablement of integrations
- Docker image build and deployment support

---

## Project layout

- [main.go](main.go) — app startup and HTTP routing
- [registry.go](registry.go) — integration wiring for file and auth services
- [auth/](auth) — auth integrations and validation logic
- [fileService/](fileService) — Nextcloud and Paperless integrations
- [shellService/](shellService) — shell execution service implementations
- [config/](config) — configuration loading and defaults
- [secretProvider/](secretProvider) — environment-based secret access
- [Dockerfile](Dockerfile) — production container image
- [docker-compose.yml](docker-compose.yml) — local compose run configuration
- [.github/workflows/deploy.yaml](.github/workflows/deploy.yaml) — self-hosted deployment workflow

---

## Requirements

- Go 1.25+
- Docker
- optional: Docker Compose for local runs
- a working Docker socket on the host if using the `dockerCmd` service

---

## Quick start

```bash
git clone https://github.com/Henriii-01/OneLab-API.git
cd OneLab-API
go mod download
cp .env.example .env
```

Then review and adjust the config in [config/config.json](config/config.json) and the environment values in [.env.example](.env.example).

---

## Configuration

The app loads configuration from two places:

1. Environment variables loaded from `.env` or the process environment
2. [config/config.json](config/config.json) for non-secret integration settings

### Auth config

The default auth config is:

```json
"auth": {
  "jwt": false,
  "oauth": true
}
```

### Service config

Example service entries in [config/config.json](config/config.json):

```json
"services": {
  "nextcloud": {
    "enabled": true,
    "url": ""
  },
  "paperless": {
    "enabled": true,
    "url": ""
  },
  "dockerCmd": {
    "enabled": true,
    "url": "local"
  }
}
```

The `dockerCmd` entry is intentionally local-only. It does not call an HTTP API and does not need a remote base URL.

The example config in [config/example-config.json](config/example-config.json) shows the same structure with disabled defaults.

---

## Environment variables

The project expects values such as these in the environment:

```env
# Auth
ONELAB_JWT_SECRET=dev-secret-change-in-production
ONELAB_AUTH_CLIENT_ID=dev-client
ONELAB_AUTH_CLIENT_SECRET=dev-password

# OAuth / OIDC auth
ONELAB_OAUTH_ISSUER_URL=https://auth.example.com/application/o/onelab/
ONELAB_OAUTH_CLIENT_ID=your-oauth-client-id
ONELAB_OAUTH_CLIENT_SECRET=your-oauth-client-secret
ONELAB_OAUTH_TOKEN_URL=https://auth.example.com/application/o/token/

# Nextcloud
ONELAB_NEXTCLOUD_BASE_URL=http://nextcloud.local/
ONELAB_NEXTCLOUD_USER=your-nextcloud-user
ONELAB_NEXTCLOUD_PASSWORD=your-nextcloud-password

# Paperless-ngx
ONELAB_PAPERLESS_BASE_URL=http://paperless.local/
ONELAB_PAPERLESS_TOKEN=your-paperless-api-token
```

### Container logging

The application writes structured JSON logs to stdout so Docker and other container runtimes can collect them. Set `ONELAB_LOG_LEVEL` to `debug`, `info` (default), `warn`, or `error` to control verbosity. The logger never writes application log files inside the container.

See [.env.example](.env.example) for the full reference file.

---

## Auth integrations

The app registers auth providers through the registry pattern in [auth/registry.go](auth/registry.go).

### `jwt`

The JWT service signs and validates tokens locally using a configured secret.

- configured via `ONELAB_JWT_SECRET`
- enabled via `config.json`
- used to issue bearer tokens for protected endpoints

### `oauth`

The OAuth integration validates bearer tokens by validating them against the configured OIDC/OAuth provider.

- configured via issuer URL and client credentials
- uses provider discovery and token introspection patterns depending on the setup

---

## File integrations

The file service integrations are registered in [fileService/registry.go](fileService/registry.go) and implemented in [fileService/nextcloud.go](fileService/nextcloud.go) and [fileService/paperless.go](fileService/paperless.go).

### `nextcloud`

- supports file lookup and file transfer
- requires `ONELAB_NEXTCLOUD_USER`
- requires `ONELAB_NEXTCLOUD_PASSWORD`
- optional base URL via `ONELAB_NEXTCLOUD_BASE_URL`

### `paperless`

- supports file lookup and transfer-style workflows for the Paperless service
- requires `ONELAB_PAPERLESS_TOKEN`
- optional base URL via `ONELAB_PAPERLESS_BASE_URL`

---

## Shell service

The shell service package is defined in [shellService/service.go](shellService/service.go) and [shellService/registry.go](shellService/registry.go).

### `dockerCmd`

The Docker command backend is implemented in [shellService/dockerCmd.go](shellService/dockerCmd.go).

It provides a service-level API for executing commands inside Docker containers using the local Docker CLI:

```go
svc, err := shellService.Builders()["dockerCmd"]()
if err != nil {
    panic(err)
}

output, err := svc.Execute(ctx, "my-container", "ls -la")
```

Important details:

- it does not expose HTTP routes
- it does not require a URL
- it relies on access to the host Docker socket
- it is intended for local container interaction from the running app

---

## HTTP API

The server is started in [main.go](main.go).

### Public routes

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | basic liveness probe |
| `POST` | `/api/v1/auth/token` | issue a bearer token from configured auth providers |

### Protected file routes

The file controller registers these routes through `authController.Middleware`:

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/api/v1/files/transfer/` | transfer a file from one configured service to another |
| `GET` | `/api/v1/files/lookup/` | search for a file in a configured source service |

Example token creation:

```bash
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"username":"<client-id>","password":"<client-secret>"}'
```

Then use the returned token in the `Authorization: Bearer <token>` header for protected requests.

---

## Running locally

### Using Go directly

```bash
go run .
```

The service listens on port `8080`.

### Using Docker Compose

This project includes [docker-compose.yml](docker-compose.yml), which builds the app and mounts the Docker socket so the shell service can reach the host Docker daemon:

```bash
docker compose up --build
```

The compose file exposes:

```yaml
ports:
  - "8080:8080"
```

and mounts:

```yaml
- /var/run/docker.sock:/var/run/docker.sock
- /etc/ssl/certs:/etc/ssl/certs:ro
```

---

## Docker image

The production image is defined in [Dockerfile](Dockerfile).

It does the following:

- builds a Go binary in a builder stage
- copies the binary into a minimal Alpine image
- installs the Docker CLI so the app can run `docker exec` commands
- exposes port `8080`

This is required because the `dockerCmd` service shells out to Docker from inside the container.

---

## Deployment

The repository includes a self-hosted GitHub Actions deployment workflow at [.github/workflows/deploy.yaml](.github/workflows/deploy.yaml).

The workflow:

- deploys to dev on the `dev` branch
- deploys to prod on the `main` branch with manual approval
- stops and removes the previous container
- rebuilds the image
- restarts the app container with environment variables and Docker socket access

The deployment also mounts the Docker socket (`/var/run/docker.sock`) so that the app can execute commands against the host engine.

---

## Security notes

- keep secrets in environment variables or a secret manager
- do not commit real credentials to source control
- restrict access to the Docker socket to trusted deployment environments only
- use strong secret values for JWT and OAuth credentials in production

---

## Known design intent

This project is intentionally modular:

- auth integrations are independent providers
- file integrations are independent providers
- shell services are independent backends
- the app selects registered implementations via config and init registration patterns

This makes it easy to add more providers without modifying the main entry point.
