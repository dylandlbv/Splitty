#!/bin/sh
set -e

/app/splitty &
APP_PID=$!

nginx -g 'daemon off;' &
NGINX_PID=$!

shutdown() {
    kill "$APP_PID" "$NGINX_PID" 2>/dev/null || true
}
trap shutdown INT TERM

(
    while kill -0 "$APP_PID" 2>/dev/null; do
        sleep 5
    done
    echo "splitty: api process exited, stopping container" >&2
    kill "$NGINX_PID" 2>/dev/null || true
) &

wait "$NGINX_PID" "$APP_PID"
