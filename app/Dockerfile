# Mehrstufiger Build: Frontend, Backend, schlankes Laufzeitbild.

FROM node:22-alpine AS frontend
WORKDIR /build
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend
WORKDIR /build
COPY app/go.mod app/go.sum* ./
RUN go mod download
COPY app/ ./
# CGO aus, damit das Binary in einem Bild ohne libc laeuft.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 healthgate
COPY --from=backend /server /usr/local/bin/server
COPY --from=frontend /build/dist /srv/web
USER healthgate
EXPOSE 8080
ENV HEALTHGATE_WEB_VERZEICHNIS=/srv/web
# Der Healthcheck des Containers nutzt denselben Endpunkt wie die Pipeline.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
	CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/server"]
