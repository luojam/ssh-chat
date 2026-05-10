FROM golang:1.26.2-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ssh-chat ./cmd/ssh-chat

FROM alpine:3.22

COPY --from=build /out/ssh-chat /usr/local/bin/ssh-chat

ENV SSH_CHAT_HOST=0.0.0.0 \
    SSH_CHAT_PORT=2222 \
    SSH_CHAT_HOST_KEY_PATH=/var/lib/ssh-chat/ssh_host_ed25519 \
    SSH_CHAT_SQLITE_PATH=/var/lib/ssh-chat/ssh-chat.sqlite

RUN mkdir -p /var/lib/ssh-chat

VOLUME ["/var/lib/ssh-chat"]
EXPOSE 2222

ENTRYPOINT ["/usr/local/bin/ssh-chat"]
