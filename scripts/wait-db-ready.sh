#!/bin/sh

: ${DB_SERVICE:=db}
: ${DB_USER:=postgres}
: ${DB_NAME:=postgres}
: ${DB_PASSWORD?required}
: ${DB_CHECK_TIMEOUT:=30}
: ${DB_CHECK_INTERVAL:=2}
: ${DOCKER_COMPOSE:=docker-compose}

check_database() {
  ${DOCKER_COMPOSE} exec -T -e PGPASSWORD="${DB_PASSWORD}" ${DB_SERVICE} \
    sh -c "pg_isready -q && psql -h ${DB_SERVICE} -U ${DB_USER} -d ${DB_NAME} -c 'SELECT 1' >/dev/null 2>&1"
}

echo "Waiting for database to be ready (timeout: ${DB_CHECK_TIMEOUT}s)..." >&2

end=$(( $(date +%s) + DB_CHECK_TIMEOUT ))
while ! check_database; do
  if [ $(date +%s) -ge $end ]; then
    echo "PostgreSQL not ready after ${DB_CHECK_TIMEOUT}s" >&2
    exit 1
  fi
  echo "Waiting ${DB_CHECK_INTERVAL}s..." >&2
  sleep $DB_CHECK_INTERVAL
done

echo "PostgreSQL is ready!" >&2
exit 0
