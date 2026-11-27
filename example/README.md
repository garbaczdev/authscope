# Example: testing authscope manually

A minimal stack that exercises authscope as the forward-auth service behind
[Caddy](https://caddyserver.com), with a mock backend that echoes the request.
See the [main README](../README.md) for what authscope is and how it works.

```
client ──▶ Caddy :4000 ──▶ authscope :8081 /verify
                     │            (200 -> attach X-Source/X-Env)
                     └──▶ whoami :80
```

- **backend** — `traefik/whoami`; prints every request header, including the
  `X-Source` and `X-Env` that authscope attached.
- **authscope** — built from the repository `Dockerfile`. Admin UI on
  `localhost:8080`, verify endpoint on `localhost:8081` (also reachable by
  Caddy on the compose network).
- **caddy** — reverse proxy on `localhost:4000`. It strips any client-supplied
  `X-Source`/`X-Env`, calls `/verify`, copies the returned identity onto the
  request, strips `Authorization`, and forwards to the backend.

## Prerequisites

- Docker with the `compose` plugin.

## Start the stack

From this directory:

```sh
docker compose up --build
```

## Create a key

1. Open the admin UI at <http://localhost:8080>.
2. Click **Create a new key**, fill in `source` and `environment` (for example
   `web` and `prod`), and save.
3. The generated token is shown **once** on the key page — copy it.

## Test it

`test.sh` sends one request through Caddy. The token can be passed as an
argument or entered interactively (hidden):

```sh
# Authenticated: expect 200 and X-Source / X-Env echoed by whoami.
./test.sh authscope-xxxxxxxxxxxxxxxx

# Interactive: prompts for the token with echo disabled.
./test.sh

# No token: sends an unauthenticated request, expect 401.
./test.sh ''
```

The response body is the backend's echo of the request headers. On success you
should see `X-Source` and `X-Env`; without a valid token Caddy returns the
`401` from authscope and never contacts the backend.

You can also call the verify endpoint directly:

```sh
curl -i -H 'Authorization: Bearer <token>' http://localhost:8081/verify
```

## Logs

Caddy logs every request to stdout, so you can watch requests and status codes
as you test:

```sh
docker compose logs -f caddy
```

## Teardown

```sh
docker compose down      # stop the stack, keep keys
docker compose down -v   # stop the stack and delete the key database
```
