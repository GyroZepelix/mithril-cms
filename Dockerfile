# Stage 1 — Build admin SPA
FROM node:20-alpine AS admin-build
WORKDIR /app/admin
COPY admin/package.json admin/package-lock.json ./
RUN npm ci
COPY admin/ ./
RUN npm run build

# Stage 2 — Build Go binary
FROM golang:1.24-alpine AS go-build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=admin-build /app/admin/dist ./admin/dist
RUN CGO_ENABLED=0 go build -tags embed_admin -o /mithril ./cmd/mithril/

# Stage 3 — Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget
COPY --from=go-build /mithril /mithril
COPY schema/ /schema/
RUN mkdir -p /data/media

# Create non-root user
RUN addgroup -S mithril && adduser -S -G mithril mithril
RUN chown -R mithril:mithril /data/media

ENV MITHRIL_SCHEMA_DIR=/schema
ENV MITHRIL_MEDIA_DIR=/data/media

EXPOSE 8080
USER mithril
ENTRYPOINT ["/mithril"]
