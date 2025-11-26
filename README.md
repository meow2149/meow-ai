# Meow-AI

End-to-end realtime voice chat powered by Doubao. The Go backend proxies WebSocket traffic to the model, while the React frontend captures microphone audio and plays streamed TTS output. This document covers building, publishing to the private registry `meow2149/meow-ai`, and operating the stack with Watchtower-based auto updates.

## Project layout

- `internal/` – Go backend (config loading, WebSocket bridge, audio pipeline)
- `web/` – React frontend source; `web/dist` is the production bundle
- `Dockerfile` – multi-stage build with `api-runtime` (Go) and `web-runtime` (Nginx)
- `compose.yaml` – local build/run using images built from source
- `compose.prod.yaml` – production override that pulls remote images + runs Watchtower
- `config.yaml` – Doubao credentials & session settings, mounted as a read-only volume

## Prerequisites

- Go ≥ 1.25.4
- pnpm ≥ 8
- Docker ≥ 24 and Docker Compose v2
- Access to Docker Hub with the private repository `meow2149/meow-ai`

## Build & publish workflow

```bash
# 1. Build the frontend bundle (required before building images)
cd web
pnpm install
pnpm run build

# 2. Build the Go + Nginx runtime images from project root
cd ..
docker compose build

# 3. Tag the freshly built images for the private repository
docker tag meow-ai-api:latest meow2149/meow-ai:api-latest
docker tag meow-ai-web:latest meow2149/meow-ai:web-latest

# Optionally version them (recommended for Watchtower)
VERSION=v1.0.0
docker tag meow-ai-api:latest meow2149/meow-ai:api-${VERSION}
docker tag meow-ai-web:latest meow2149/meow-ai:web-${VERSION}

# 4. Push
docker push meow2149/meow-ai:api-latest
docker push meow2149/meow-ai:web-latest
docker push meow2149/meow-ai:api-${VERSION}
docker push meow2149/meow-ai:web-${VERSION}
```

> Use semantic tags (`api-v1.2.0`, `web-v1.2.0`, etc.) so Watchtower can pick up new releases automatically.

## Docker login (workstation & server)

```bash
# Interactive login
docker login -u meow2149

# Non-interactive (CI / remote shell)
echo "$DOCKERHUB_TOKEN" | docker login -u meow2149 --password-stdin
```

Run the same command on the server before pulling or starting the stack so the host can access the private repository.

## Local development run

```bash
# Build + start using locally built images
docker compose up -d --build

# Health checks
docker compose ps
docker compose logs -f api
docker compose logs -f web
```

Stop everything with `docker compose down`.

## Production deployment

1. Copy a production-ready `config.yaml` next to the compose files on the server.
1. Log in to Docker Hub on the server (`docker login -u meow2149`).
1. Point Compose to the private images (override defaults if necessary):

```bash
export MEOW_AI_API_IMAGE=meow2149/meow-ai:api-latest
export MEOW_AI_WEB_IMAGE=meow2149/meow-ai:web-latest
# Optional: change exposed HTTP port
export MEOW_AI_WEB_PORT=8080
```

1. Start everything with the production override:

```bash
docker compose -f compose.yaml -f compose.prod.yaml up -d
```

`compose.prod.yaml` adds:

- `pull_policy: always` so containers pull newer tags when starting.
- `meow-ai-watchtower`, polling every 300 seconds and restarting `meow-ai-api` / `meow-ai-web` when fresh tags appear.

## Auto-update flow

1. Build and push new images (`api` + `web`) to `meow2149/meow-ai` with new tags.
1. Watchtower on the server notices the new tags during the next poll and automatically pulls + restarts the affected containers.
1. To force an immediate update, run:

```bash
docker exec meow-ai-watchtower watchtower --run-once meow-ai-api meow-ai-web
```

Tune the polling interval via `WATCHTOWER_POLL_INTERVAL` if needed.

## Useful commands

```bash
# Stop everything (including Watchtower)
docker compose -f compose.yaml -f compose.prod.yaml down

# Restart only API or web
docker restart meow-ai-api
docker restart meow-ai-web

# Tail logs
docker logs -f meow-ai-api
docker logs -f meow-ai-web

# Cleanup (remove volumes & dangling images)
docker compose down -v
docker image prune
```

## Notes & best practices

- `config.yaml` contains secrets; keep it outside of the images and mount it read-only (`./config.yaml:/config/config.yaml:ro`).
- Always rebuild `web/dist` before creating a new `web-runtime` image, otherwise old assets remain.
- Customize Nginx behavior by editing `nginx.conf` and rebuilding the `web-runtime` target.
- When using custom tags, remember to update `MEOW_AI_API_IMAGE` / `MEOW_AI_WEB_IMAGE` on the server so Watchtower tracks the right references.
