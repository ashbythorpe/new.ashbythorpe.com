#!/bin/sh
set -e

if [ -f /data/app.db ]; then
    echo "Database already exists, skipping restore"
else
    echo "No database found, restoring from replica if exists"
    litestream restore -v -if-replica-exists -o /data/app.db "${LITESTREAM_REPLICA_URL}" || true
fi

exec litestream replicate -exec "run-app"
