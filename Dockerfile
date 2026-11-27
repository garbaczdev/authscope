# syntax=docker/dockerfile:1

# --- build -----------------------------------------------------------------
FROM golang:1.27-alpine AS build

# mattn/go-sqlite3 is cgo, so the C toolchain is needed at build time.
RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/authscope ./cmd/authscope

# --- runtime ---------------------------------------------------------------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 authscope

WORKDIR /app
COPY --from=build /out/authscope /usr/local/bin/authscope
RUN mkdir -p /app/data && chown -R authscope:authscope /app
USER authscope

EXPOSE 8080 8081
ENTRYPOINT ["authscope"]
