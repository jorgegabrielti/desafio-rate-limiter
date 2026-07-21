# Estágio 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Instala ca-certificates
RUN apk add --no-cache git ca-certificates

# Copia arquivos de dependência
COPY go.mod go.sum ./
RUN go mod download

# Copia o código fonte
COPY . .

# Compila o binário estático
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server/main.go

# Estágio 2: Runner final enxuto
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/server /app/server
COPY --from=builder /app/.env /app/.env

EXPOSE 8080

ENTRYPOINT ["/app/server"]
