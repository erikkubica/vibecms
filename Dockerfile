FROM node:20-alpine AS frontend
WORKDIR /app/admin-ui
COPY admin-ui/package*.json ./
RUN npm ci
COPY admin-ui/ .
RUN npm run build

# Build extension admin UIs
WORKDIR /app/extensions/media-manager/admin-ui
COPY extensions/media-manager/admin-ui/package*.json ./
RUN npm ci
COPY extensions/media-manager/admin-ui/ .
RUN npm run build

WORKDIR /app/extensions/email-manager/admin-ui
COPY extensions/email-manager/admin-ui/package*.json ./
RUN npm ci
COPY extensions/email-manager/admin-ui/ .
RUN npm run build

FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o vibecms ./cmd/vibecms
# Build extension plugin binaries
RUN CGO_ENABLED=0 go build -o extensions/smtp-provider/bin/smtp-provider ./extensions/smtp-provider/cmd/plugin/
RUN CGO_ENABLED=0 go build -o extensions/sitemap-generator/bin/sitemap-generator ./extensions/sitemap-generator/cmd/plugin/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/vibecms .
COPY --from=builder /app/ui/templates ./ui/templates
COPY --from=builder /app/themes ./themes
COPY --from=builder /app/extensions ./extensions
COPY --from=frontend /app/admin-ui/dist ./admin-ui/dist
COPY --from=frontend /app/extensions/media-manager/admin-ui/dist ./extensions/media-manager/admin-ui/dist
COPY --from=frontend /app/extensions/email-manager/admin-ui/dist ./extensions/email-manager/admin-ui/dist
EXPOSE 8099
CMD ["./vibecms"]
