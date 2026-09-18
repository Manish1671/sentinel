# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY packages/telemetry /src/packages/telemetry
COPY services/remediation/go.mod services/remediation/go.sum /src/services/remediation/
WORKDIR /src/services/remediation
RUN go mod download
COPY services/remediation /src/services/remediation
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/remediation ./cmd/remediation

FROM alpine:3.20
RUN adduser -D -H -u 65532 sentinel
WORKDIR /app
COPY --from=build /out/remediation /app/remediation
COPY database/migrations /app/database/migrations
USER 65532:65532
ENV MIGRATIONS_PATH=/app/database/migrations
EXPOSE 8093
ENTRYPOINT ["/app/remediation"]
