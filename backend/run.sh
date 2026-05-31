#!/bin/sh
set -e

rm -f /data/app.db /data/app.db-wal /data/app.db-shm

# if [ -f /data/app.db ]; then
#     echo "Database already exists, skipping restore"
# else
#     echo "No database found, restoring from replica if exists"
#     litestream restore -v -if-replica-exists -o /data/app.db "${LITESTREAM_REPLICA_URL}" || true
# fi

cloudflared tunnel --no-autoupdate run --token ${CLOUDFLARE_TUNNEL_TOKEN} &

exec litestream replicate -exec "run-app"
