# Meow-AI

Meow-AI is an intelligent real-time voice companion designed to provide empathetic listening and emotional support. It creates a safe, judgment-free space for users to share their thoughts and feelings, serving as a supportive partner for mental well-being.

## Local Development

```bash
# 1. Build frontend assets
cd web
pnpm install
pnpm run build

# 2. Build and start services
cd ..
docker compose up -d --build
```

## Release Workflow

```bash
# 1. Build frontend
cd web
pnpm install
pnpm run build

# 2. Build Docker images
cd ..
docker compose build

# 3. Tag images
docker tag meow-ai-api:latest meow2149/meow-ai:api-latest
docker tag meow-ai-web:latest meow2149/meow-ai:web-latest

# 4. Push to registry
docker push meow2149/meow-ai:api-latest
docker push meow2149/meow-ai:web-latest
```

## Production Deployment

```bash
# 1. Prepare configuration
# Copy config.yaml, compose.yaml, and compose.prod.yaml to the server

# 2. Authenticate
docker login -u meow2149

# 3. Configure environment
export MEOW_AI_API_IMAGE=meow2149/meow-ai:api-latest
export MEOW_AI_WEB_IMAGE=meow2149/meow-ai:web-latest

# 4. Launch services
docker compose -f compose.yaml -f compose.prod.yaml up -d
```
