# --- Go API build stage -------------------------------------------------------
FROM golang:1.25.4-alpine AS api-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /out/meow-ai .

# --- Go API runtime stage -----------------------------------------------------
FROM gcr.io/distroless/base-debian12:nonroot AS api-runtime

WORKDIR /app

COPY --from=api-builder /out/meow-ai /usr/local/bin/meow-ai

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/meow-ai"]

# --- Nginx runtime stage ------------------------------------------------------
FROM nginx:1.27-alpine AS web-runtime

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY web/dist /usr/share/nginx/html
