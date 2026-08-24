# Build stage
FROM golang:1.26-alpine@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

# Run stage
#
# Node, rather than bare alpine, because AI signal generation shells out
# to the Claude Code CLI. That runs on the owner's subscription instead
# of the metered API, and the CLI needs a Node runtime. The Go binary is
# static (CGO disabled), so it runs here unchanged.
FROM node:20-alpine@sha256:fb4cd12c85ee03686f6af5362a0b0d56d50c58a04632e6c0fb8363f609372293

WORKDIR /app

RUN npm install -g @anthropic-ai/claude-code && npm cache clean --force

# The CLI needs a writable config directory; /root is not guaranteed to
# be one on every host.
ENV CLAUDE_CONFIG_DIR=/app/.claude
RUN mkdir -p /app/.claude

COPY --from=builder /app/main .
COPY --from=builder /app/.env* ./

EXPOSE 3000

CMD ["./main"]
