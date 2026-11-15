# Build stage
FROM golang:1.26-alpine@sha256:30b1e546a9c7fa3b747c2bdba3a0c7bdf3a02ebb4c5b8405d2f948d7e8aac4ec AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go \
 && CGO_ENABLED=0 GOOS=linux go build -o backfill ./cmd/backfill

# Run stage
#
# Node, rather than bare alpine, because AI signal generation shells out
# to the Claude Code CLI. That runs on the owner's subscription instead
# of the metered API, and the CLI needs a Node runtime. The Go binary is
# static (CGO disabled), so it runs here unchanged.
FROM node:24-alpine@sha256:116b800c3754b440ee0aa3ed2f20f5bf3d99a554bfd9adeea09ea8017f6e85f5

WORKDIR /app

# Pinned: an unpinned install is frozen by the build cache at whatever
# version first populated it, and 2.1.197 was still being served on
# 2026-10-10 although it cannot run claude-opus-5-5 (needs >= 2.1.280).
# Bump deliberately when AI_MODEL moves to a newer model.
ARG CLAUDE_CODE_VERSION=2.1.296
# npm is only needed to install the CLI. Removing it afterwards drops its
# bundled dependencies (tar, glob, minimatch, ...), which were every
# node-pkg finding in the nightly Trivy scan. apk upgrade picks up Alpine
# fixes published after the pinned base digest.
RUN apk upgrade --no-cache  && npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}  && npm cache clean --force  && rm -rf /usr/local/lib/node_modules/npm /usr/local/bin/npm /usr/local/bin/npx

# The CLI needs a writable config directory; /root is not guaranteed to
# be one on every host.
ENV CLAUDE_CONFIG_DIR=/app/.claude
RUN mkdir -p /app/.claude

COPY --from=builder /app/main .
# One-off history import; see cmd/backfill. Not run by default.
COPY --from=builder /app/backfill .
COPY --from=builder /app/.env* ./

EXPOSE 3000

CMD ["./main"]
