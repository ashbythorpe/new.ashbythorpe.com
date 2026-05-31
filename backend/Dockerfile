ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /run-app .


FROM debian:bookworm

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=litestream/litestream:latest /usr/local/bin/litestream /usr/local/bin/litestream

COPY --from=cloudflare/cloudflared:latest /usr/local/bin/cloudflared /usr/local/bin/cloudflared

WORKDIR /app

COPY --from=builder /run-app /usr/local/bin/

COPY litestream.yml /etc/litestream.yml
COPY run.sh /app/run.sh

RUN chmod +x /app/run.sh

EXPOSE 3000

CMD ["/app/run.sh"]
