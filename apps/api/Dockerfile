# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY packages/telemetry /src/packages/telemetry
COPY apps/api/go.mod apps/api/go.sum /src/apps/api/
WORKDIR /src/apps/api
RUN go mod download
COPY apps/api /src/apps/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.20
RUN adduser -D -H -u 65532 sentinel
WORKDIR /app
COPY --from=build /out/api /app/api
COPY database/migrations /app/database/migrations
USER 65532:65532
ENV MIGRATIONS_PATH=/app/database/migrations
EXPOSE 8080
ENTRYPOINT ["/app/api"]
