FROM node:20-alpine AS frontend
# Build admin UI
WORKDIR /app/admin-ui
COPY admin-ui/package*.json ./
RUN npm ci
COPY admin-ui/ .
RUN node scripts/generate-icon-shim.cjs && npm run build

# Build all extension admin UIs that have a package.json
COPY extensions/ /app/extensions/
RUN for dir in /app/extensions/*/admin-ui; do \
      [ -f "$dir/package.json" ] || continue; \
      echo "Building $dir..."; \
      cd "$dir" && npm ci && npm run build && cd /app; \
    done

FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build core
RUN CGO_ENABLED=0 go build -o vibecms ./cmd/vibecms
# Build all extension plugins that have a cmd/plugin/main.go
RUN for dir in extensions/*/cmd/plugin; do \
      [ -f "$dir/main.go" ] || continue; \
      slug=$(echo "$dir" | cut -d/ -f2); \
      echo "Building plugin $slug..."; \
      CGO_ENABLED=0 go build -o "extensions/$slug/bin/$slug" "./$dir/"; \
    done

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/vibecms .
COPY --from=builder /app/ui/templates ./ui/templates
COPY --from=builder /app/themes ./themes
COPY --from=builder /app/extensions ./extensions
COPY --from=frontend /app/admin-ui/dist ./admin-ui/dist
COPY --from=frontend /app/extensions/ ./extensions/
EXPOSE 8099
CMD ["./vibecms"]
