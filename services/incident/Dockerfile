# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY services/incident/go.mod services/incident/go.sum ./
RUN go mod download
COPY services/incident ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/incident ./cmd/incident

FROM alpine:3.20
RUN adduser -D -H -u 65532 sentinel
WORKDIR /app
COPY --from=build /out/incident /app/incident
COPY database/migrations /app/database/migrations
COPY database/seeds /app/database/seeds
USER 65532:65532
ENV MIGRATIONS_PATH=/app/database/migrations
ENV SEEDS_PATH=/app/database/seeds
EXPOSE 8092
ENTRYPOINT ["/app/incident"]
