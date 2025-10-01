FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/download-accelerator .

FROM alpine:3.20
WORKDIR /app
ENV GIN_MODE=release
COPY --from=builder /out/download-accelerator /usr/local/bin/download-accelerator
COPY configs/ configs/
COPY web/ web/
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/download-accelerator"]

