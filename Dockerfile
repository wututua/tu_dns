# Stage 1 — build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2 — build Go binary with embedded SPA
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY --from=frontend /app/frontend/dist /app/webembed/dist
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /app/bin/tudns .

# Stage 3 — minimal runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -h /data tudns
COPY --from=backend /app/bin/tudns /usr/local/bin/tudns
USER tudns
WORKDIR /data
EXPOSE 8080
ENV TUDNS_CONFIG=/data/config.yaml
CMD ["tudns"]
