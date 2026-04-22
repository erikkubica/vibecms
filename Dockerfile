FROM node:20-alpine AS frontend
WORKDIR /app/admin-ui
COPY admin-ui/package*.json ./
RUN npm ci
COPY admin-ui/ .
RUN node scripts/generate-icon-shim.cjs && npm run build

# Build all extension admin-UI micro-frontends
FROM node:20-alpine AS ext-frontend
WORKDIR /app
# Copy all extension admin-ui source at once
COPY extensions/ /tmp/extensions/
RUN for dir in /tmp/extensions/*/admin-ui; do \
      [ -f "$dir/package.json" ] || continue; \
      slug=$(basename $(dirname "$dir")); \
      echo "Building extension admin-ui: $slug"; \
      mkdir -p "/app/extensions/$slug/admin-ui"; \
      cp -r "$dir/." "/app/extensions/$slug/admin-ui/"; \
      cd "/app/extensions/$slug/admin-ui"; \
      npm install --production=false 2>/dev/null; \
      npm run build 2>/dev/null; \
      rm -rf node_modules src; \
    done

FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build core
RUN CGO_ENABLED=0 go build -o vibecms ./cmd/vibecms
# Build all extension plugins
RUN for dir in extensions/*/cmd/plugin; do \
      [ -f "$dir/main.go" ] || continue; \
      slug=$(echo "$dir" | cut -d/ -f2); \
      echo "Building plugin: $slug"; \
      CGO_ENABLED=0 go build -o "extensions/$slug/bin/$slug" "./$dir/"; \
    done

FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
    jpegoptim optipng pngquant imagemagick ffmpeg
WORKDIR /app
COPY --from=builder /app/vibecms .
COPY --from=builder /app/ui/templates ./ui/templates
COPY --from=builder /app/themes ./themes
COPY --from=builder /app/extensions ./extensions
COPY --from=frontend /app/admin-ui/dist ./admin-ui/dist
# Overlay extension admin-ui dist files (replaces the Go-stage copies with built versions)
COPY --from=ext-frontend /app/extensions/ ./extensions/
EXPOSE 8099
CMD ["./vibecms"]
