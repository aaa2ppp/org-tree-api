#!/bin/sh

redefined_db_addr=$DB_ADDR

. ./.env

DB_ADDR=${redefined_db_addr:-$DB_ADDR}
MIGRATIONS_DIR=${MIGRATIONS_DIR:-"./migrations"}

if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo "$(basename $0): $MIGRATIONS_DIR dir not found" >&2
    exit 1
fi

host=${DB_ADDR%:*}
port=${DB_ADDR#*:}
[ "$port" = "$host" ] && port=

: ${host:=localhost}
: ${port:=5432}

dbname=${DB_NAME:-postgres}
user=${DB_USER:-postgres}
password=${DB_PASSWORD?required}
sslmode=${DB_SSLMODE:-disable}

export PGPASSWORD=$password
export GOOSE_DBSTRING="host=$host port=$port dbname=$dbname user=$user sslmode=$sslmode"
goose -dir "$MIGRATIONS_DIR" postgres "$@"
