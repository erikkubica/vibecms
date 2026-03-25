FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o vibecms ./cmd/vibecms

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/vibecms .
COPY --from=builder /app/ui/templates ./ui/templates
EXPOSE 8099
CMD ["./vibecms"]
