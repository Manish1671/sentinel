# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY packages/telemetry /src/packages/telemetry
COPY services/ingestion/go.mod services/ingestion/go.sum /src/services/ingestion/
WORKDIR /src/services/ingestion
RUN go mod download
COPY services/ingestion /src/services/ingestion
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/ingestion ./cmd/ingestion

FROM alpine:3.20
RUN adduser -D -H -u 65532 sentinel
WORKDIR /app
COPY --from=build /out/ingestion /app/ingestion
USER 65532:65532
EXPOSE 8090
ENTRYPOINT ["/app/ingestion"]
