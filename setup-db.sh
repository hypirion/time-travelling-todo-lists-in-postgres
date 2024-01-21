#!/usr/bin/env bash

set -euo pipefail

pgpass=mySecretPassword
pgport=${POSTGRES_PORT:-10840}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

if docker ps -a --filter="name=temporal-table-test-db" --format "{{.Names}}" | \
        grep -q 'temporal-table-test-db'; then
    echo "Stopping old container"
    docker stop temporal-table-test-db > /dev/null
fi

echo "Starting container"
docker run --rm -d \
       --name temporal-table-test-db \
       --tmpfs=/var/lib/postgresql/data \
       -e POSTGRES_PASSWORD="${pgpass}" \
       -p "${pgport}:5432" \
       -d postgres:16 \
       -c fsync=off

if command -v pg_isready &> /dev/null; then
    countdown=10
    while ! pg_isready -h localhost -p "${pgport}" >& /dev/null; do
        printf "\r%02d" ${countdown}
        sleep 1
        countdown=$((countdown-1))
        if [ ${countdown} -le 0 ]; then
            echo -e "\rTook more than 10 seconds to wait for database to come up."
            echo "This seems suspiciously long, so I'll rather die than let you wait."
            exit 1
        fi
    done
    echo -e '\rTest database ready'
else
    sleep 2
    echo 'pg_isready not found, but database should be ready now'
fi
