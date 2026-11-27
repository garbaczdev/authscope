#!/usr/bin/env bash
set -euo pipefail

# Sends one request through Caddy (localhost:4000) to the mock backend and
# prints the full response, including any identity headers.
#
# Usage: ./test.sh [token]
#   token given        -> authenticated request (expect 200 + X-Source/X-Env)
#   no token, empty    -> unauthenticated request (expect 401)
#   no token, no TTY   -> unauthenticated request (no prompt)

token="${1:-}"

# Only prompt when there is no argument and stdin is a terminal.
if [ -z "$token" ] && [ -t 0 ]; then
	read -r -s -p 'Token (leave empty to test unauthenticated): ' token
	printf '\n' >&2
fi

if [ -n "$token" ]; then
	curl -sS -i -H "Authorization: Bearer ${token}" http://localhost:4000/
else
	printf 'no token provided - sending unauthenticated request (expect 401)\n' >&2
	curl -sS -i http://localhost:4000/
fi
