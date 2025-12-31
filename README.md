# [Meow-AI](http://meow-ai.maojiucloud.cn/)

Meow-AI is an intelligent real-time voice companion designed to provide empathetic listening and emotional support. It creates a safe, judgment-free space for users to share their thoughts and feelings, serving as a supportive partner for mental well-being.

## Doubao Realtime Speech API

> [!NOTE]
> Realtime voice is powered by Volcengine Doubao end-to-end speech model.  
> Docs: [Doubao End-to-End Realtime Speech API](https://www.volcengine.com/docs/6561/1594360?lang=zh).

## Configuration

Before running the project, configure `config.yaml` with your Volcengine credentials:

```yaml
api:
  url: wss://openspeech.bytedance.com/api/v3/realtime/dialogue
  app_id: YOUR_APP_ID        # Your APP ID from Volcengine console
  app_key: PlgvMymc7f3tQnJ6
  resource_id: volc.speech.dialog
  access_key: YOUR_ACCESS_TOKEN  # Your Access Token from Volcengine console
```

## Local Development

> [!TIP]
> For local development, run the backend with Go directly and use pnpm dev server for the frontend.

```bash
# Start backend (in project root)
go mod download
go run .

# Start frontend (in web directory)
cd web
pnpm install
pnpm dev
```

## Local Deployment

For local deployment using Docker Compose:

```bash
# 1. Build frontend assets
cd web
pnpm install
pnpm build

# 2. Start services with Docker Compose
cd ..
docker compose -f compose.dev.yaml up -d
```

The services will build Docker images locally and start containers. The web interface will be available at `http://localhost`.

## Release Workflow

Build Docker images and push to Docker Hub:

```bash
# 1. Build frontend assets
cd web
pnpm install
pnpm build

# 2. Build Docker images
cd ..
docker compose -f compose.dev.yaml build

# 3. Tag images
docker tag meow-ai-api:latest meow2149/meow-ai:api-latest
docker tag meow-ai-web:latest meow2149/meow-ai:web-latest

# 4. Push to Docker Hub
docker push meow2149/meow-ai:api-latest
docker push meow2149/meow-ai:web-latest
```

## Production Deployment

Deploy on production server:

```bash
# 1. Configure domain on Cloudflare and enable proxy

# 2. Prepare configuration files
# Copy config.yaml and compose.yaml to the server

# 3. Start services
docker compose up -d
```

Services will automatically pull the latest public images from Docker Hub and start. Watchtower will automatically monitor and update containers.

## License

Meow-AI is [MIT licensed](./LICENSE).
