FROM golang:1.26 AS builder
WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /bin/task-api .

FROM scratch

COPY --from=builder /bin/task-api /task-api

USER 65534:65534

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=2s --start-period=2s --retries=3 \
    CMD ["/task-api", "healthcheck"]

CMD ["/task-api"]
