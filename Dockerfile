# Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

COPY web_client/package*.json ./
RUN npm ci

COPY web_client/ ./
RUN npm run build

# Build backend
FROM golang:1.21-alpine AS backend-builder

WORKDIR /app

RUN apk add --no-cache git

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server main.go

# Final stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=backend-builder /server /app/server
COPY --from=frontend-builder /frontend/dist /app/static

EXPOSE 8080

CMD ["/app/server"]
