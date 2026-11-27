# authscope

Forward-auth service for a reverse proxy. It decides whether an incoming request is authorized. If it is, it returns the identity of the sender, and the proxy attaches that identity before forwarding the request upstream.

It also ships a small web admin UI for managing access keys.

## How it works

1. The reverse proxy calls `GET /verify` on the auth service for each incoming request, passing only the original headers.
2. The auth service checks the bearer token against its in-memory key set.
3. On `200`, the proxy copies `X-Source` and `X-Env` onto the request, strips `Authorization`, and forwards it. On anything else, the request is denied.

## CLI options

Every option can be set as a command-line flag or an environment variable. The flag wins when both are given; otherwise the env var is used, falling back to the default.

| Flag | Environment variable | Default | Description |
|---|---|---|---|
| `-admin-addr` | `ADMIN_ADDR` | `0.0.0.0:8080` | Address to expose the admin panel on. |
| `-verify-addr` | `VERIFY_ADDR` | `0.0.0.0:8081` | Address to expose the verify endpoint on. |
| `-data-dir` | `DATA_DIR` | `./data` | Directory authscope stores its data in (created if missing). |

## Endpoints

### `GET /verify` (internal network only, plain HTTP)

Reads `Authorization: Bearer <token>`. All other headers are ignored.

| Status | When | Response headers |
|---|---|---|
| `200 OK` | Token valid | `X-Source`, `X-Env` |
| `401 Unauthorized` | Header missing/malformed, or token unknown/revoked | `WWW-Authenticate: Bearer` |
| `5xx` | Internal failure | none |

The body is always empty.

### `GET /healthz`

Returns `200 OK` when the service is ready to answer `/verify`.

### Admin UI

Plain HTTP, form-based, single HTML file, minimal JS. No frontend build. Actions: create key (generated server-side), edit, delete, toggle active. The generated token is shown once, right after creation.

## Reverse-proxy requirements

- Strip any client-supplied `X-Source` / `X-Env` before calling `/verify`.
- Treat timeouts and `5xx` as denial (fail closed).
- Never forward a request that did not get `2xx`.

## Guarantees

- **Stateless:** no sessions or cookies; every request is verified independently.
- **Fast:** in-memory decision, no external calls.
- **No secret leakage:** responses never echo the token or any part of it.
- **Live reload:** adding or revoking a key takes effect without a restart; in-flight requests are unaffected.
- **Fail-safe:** an invalid config update is rejected and the previous valid config stays active.

## Example

See [`example/README.md`](example/README.md) for a runnable Caddy + whoami stack that exercises authscope end to end.
