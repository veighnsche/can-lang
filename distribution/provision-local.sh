#!/bin/sh
# Local live-dependency provisioning for H02 (databases), H03 (object storage),
# and C01 (container Firefox runner). Idempotent: safe to re-run; existing
# containers, volumes, and secrets are reused.
# Secrets live ONLY in $PROVISION_DIR (default $HOME/.can-provision, mode 700/600)
# and in process env. Nothing here prints or stores a credential value in the repo.
#
# Usage:
#   provision-local.sh db up        # start PG + MySQL, create DBs/users
#   provision-local.sh db exports   # print export lines (eval in operator shell)
#   provision-local.sh db mkdb <name> [pg|mysql]   # mint one isolated database
#   provision-local.sh db down      # stop containers (data volumes kept)
#   provision-local.sh s3 up        # start MinIO, ensure bucket
#   provision-local.sh s3 exports   # print export lines (eval in operator shell)
#   provision-local.sh s3 down      # stop container (data volume kept)
#   provision-local.sh browser up   # start the container Firefox ws server
#   provision-local.sh browser exports  # print export lines (eval in operator shell)
#   provision-local.sh browser down # stop container (data volume kept)
set -eu

PROVISION_DIR="${CAN_PROVISION_DIR:-$HOME/.can-provision}"
PG_CONTAINER=can-pg17
MYSQL_CONTAINER=can-mysql84
MINIO_CONTAINER=can-minio
FF_CONTAINER=can-ff
PG_IMAGE='postgres:17@sha256:f4c66b820c6f974249089d3d16d86a3698eae11e8746eb6644b2271031e91232'
MYSQL_IMAGE='mysql:8.4@sha256:0744ee5ef89ce6ccfa13de3e579fe6b9e27f93dd70da9c06d2c908b1b193fb8d'
MINIO_IMAGE='quay.io/minio/minio@sha256:14cea493d9a34af32f524e538b8346cf79f3321eff8e708c1e2960462bd8936e'
FF_IMAGE='mcr.microsoft.com/playwright:v1.55.1-noble@sha256:2f29369043d81d6d69a815ceb80760f55e85f5020371ad06a4d996f18503ad1c'
PG_PORT=55433
MYSQL_PORT=3307
S3_PORT=9000
S3_BUCKET_DEFAULT=can-b1-10
FF_PORT=18783
# Pinned launchServer entry: serves one Firefox 141 over ws on argv[2] and
# prints its tokened endpoint. Static text; the port arrives as an argument.
FF_SERVER_JS='import { firefox } from "playwright";
const port = Number(process.argv[2]);
const server = await firefox.launchServer({ port, timeout: 120000 });
console.log("WS=" + server.wsEndpoint());
await new Promise(() => {});
'

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1" >&2; exit 1; }; }

secrets_init() {
  mkdir -p "$PROVISION_DIR"
  chmod 700 "$PROVISION_DIR"
  for f in pg.pw mysql.pw mysql-app.pw s3.access s3.secret; do
    if [ ! -f "$PROVISION_DIR/$f" ]; then
      case "$f" in
        # Fixed by the committed live harness, not chosen here:
        # runtime/test/mysql.test.ts derives its refused/bad-password/
        # absent-db legs by rewriting ":3307/", ":<pw>@" and "/can_b1_03"
        # inside CAN_TEST_MYSQL_URL, so the provisioned URL must use
        # port 3307, database can_b1_03, and this password.
        mysql-app.pw) printf '%s' 'test-pw' >"$PROVISION_DIR/$f" ;;
        s3.access) openssl rand -hex 10 >"$PROVISION_DIR/$f" ;;
        s3.secret) openssl rand -base64 30 | tr -d '\n/' >"$PROVISION_DIR/$f" ;;
        *) openssl rand -base64 24 >"$PROVISION_DIR/$f" ;;
      esac
      chmod 600 "$PROVISION_DIR/$f"
    fi
  done
}

container_up() { # name image ports docker-args... (optional image cmd via $CMD_ARGS)
  name="$1"; image="$2"; ports="$3"; shift 3
  if docker inspect "$name" >/dev/null 2>&1; then
    docker start "$name" >/dev/null
  else
    # shellcheck disable=SC2086
    docker run -d --name "$name" --platform linux/arm64 $ports "$@" "$image" ${CMD_ARGS:-} >/dev/null
  fi
}

db_wait_pg() {
  for _ in $(seq 1 24); do
    if PGPASSWORD="$(cat "$PROVISION_DIR/pg.pw")" pg_isready -h 127.0.0.1 -p "$PG_PORT" -U postgres >/dev/null 2>&1; then return 0; fi
    sleep 5
  done
  echo "postgres did not become ready" >&2; return 1
}

db_wait_mysql() {
  for _ in $(seq 1 24); do
    if docker exec -e MYSQL_PWD="$(cat "$PROVISION_DIR/mysql.pw")" "$MYSQL_CONTAINER" mysqladmin -uroot ping >/dev/null 2>&1; then return 0; fi
    sleep 5
  done
  echo "mysql did not become ready" >&2; return 1
}

mkdb_pg() {
  PGPASSWORD="$(cat "$PROVISION_DIR/pg.pw")" psql -h 127.0.0.1 -p "$PG_PORT" -U postgres -c "CREATE DATABASE \"$1\";" >/dev/null
}
mkdb_mysql() {
  tmp="$(mktemp)"; chmod 600 "$tmp"
  printf 'CREATE DATABASE IF NOT EXISTS `%s`;' "$1" >"$tmp"
  docker exec -i -e MYSQL_PWD="$(cat "$PROVISION_DIR/mysql.pw")" "$MYSQL_CONTAINER" mysql -uroot <"$tmp" >/dev/null
  rm -f "$tmp"
}

db_up() {
  need docker; need pg_isready; need psql; need openssl
  secrets_init
  container_up "$PG_CONTAINER" "$PG_IMAGE" "-p 127.0.0.1:${PG_PORT}:5432" \
    -e POSTGRES_PASSWORD_FILE=/run/secrets/pg.pw \
    -v "$PROVISION_DIR/pg.pw:/run/secrets/pg.pw:ro" \
    -v can-pg17-data:/var/lib/postgresql/data --restart unless-stopped
  container_up "$MYSQL_CONTAINER" "$MYSQL_IMAGE" "-p 127.0.0.1:${MYSQL_PORT}:3306" \
    -e MYSQL_ROOT_PASSWORD_FILE=/run/secrets/mysql.pw \
    -v "$PROVISION_DIR/mysql.pw:/run/secrets/mysql.pw:ro" \
    -v can-mysql84-data:/var/lib/mysql --restart unless-stopped
  db_wait_pg
  db_wait_mysql
  for db in can_e02 can_f02 can_f04 can_f06; do
    PGPASSWORD="$(cat "$PROVISION_DIR/pg.pw")" psql -h 127.0.0.1 -p "$PG_PORT" -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$db';" | grep -q 1 || mkdb_pg "$db"
  done
  tmp="$(mktemp)"; chmod 600 "$tmp"
  {
    echo 'CREATE DATABASE IF NOT EXISTS can_b1_03;'
    echo 'CREATE DATABASE IF NOT EXISTS can_f02;'
    echo 'CREATE DATABASE IF NOT EXISTS can_f04;'
    echo 'CREATE DATABASE IF NOT EXISTS can_f06;'
    printf "CREATE USER IF NOT EXISTS 'canapp'@'%%' IDENTIFIED BY '%s';\n" "$(cat "$PROVISION_DIR/mysql-app.pw")"
    echo 'GRANT ALL PRIVILEGES ON `can\_%`.* TO '"'"'canapp'"'"'@'"'"'%'"'"';'
    echo 'GRANT ALL PRIVILEGES ON `can_b1_03`.* TO '"'"'canapp'"'"'@'"'"'%'"'"';'
    echo 'FLUSH PRIVILEGES;'
  } >"$tmp"
  docker exec -i -e MYSQL_PWD="$(cat "$PROVISION_DIR/mysql.pw")" "$MYSQL_CONTAINER" mysql -uroot <"$tmp" >/dev/null
  rm -f "$tmp"
  echo "db up: postgres 127.0.0.1:$PG_PORT, mysql 127.0.0.1:$MYSQL_PORT"
}

urlencode() { python3 -c 'import sys,urllib.parse; print(urllib.parse.quote(sys.stdin.read().strip(), safe=""))'; }

db_exports() {
  pg_pw="$(cat "$PROVISION_DIR/pg.pw" | urlencode)"
  app_pw="$(cat "$PROVISION_DIR/mysql-app.pw" | urlencode)"
  echo "export CAN_TEST_POSTGRES_URL=\"postgres://postgres:${pg_pw}@127.0.0.1:${PG_PORT}/<db>\""
  echo "export CAN_TEST_MYSQL_URL=\"mysql://canapp:${app_pw}@127.0.0.1:${MYSQL_PORT}/<db>\""
  echo "export CAN_TEST_MYSQL_CONTAINER=\"$MYSQL_CONTAINER\""
  echo "# Replace <db> with the isolated per-run database (see 'db mkdb')."
}

db_mkdb() {
  name="${1:?usage: db mkdb <name> [pg|mysql]}"; scope="${2:-both}"
  case "$name" in can_*_run*|can_probe_*) ;; *) echo "refusing non-isolated name (use can_<lane>_run<k> or can_probe_<...>)" >&2; exit 1 ;; esac
  case "$scope" in pg|both) mkdb_pg "$name"; echo "pg: $name" ;; esac
  case "$scope" in mysql|both) mkdb_mysql "$name"; echo "mysql: $name" ;; esac
}

db_down() { docker stop "$PG_CONTAINER" "$MYSQL_CONTAINER" >/dev/null 2>&1 || true; echo "db stopped"; }

s3_up() {
  need docker; need bun; need openssl
  secrets_init
  CMD_ARGS='server /data --console-address :9001'
  container_up "$MINIO_CONTAINER" "$MINIO_IMAGE" "-p 127.0.0.1:${S3_PORT}:9000" \
    -e MINIO_ROOT_USER_FILE=/run/secrets/s3.access \
    -e MINIO_ROOT_PASSWORD_FILE=/run/secrets/s3.secret \
    -v "$PROVISION_DIR/s3.access:/run/secrets/s3.access:ro" \
    -v "$PROVISION_DIR/s3.secret:/run/secrets/s3.secret:ro" \
    -v can-minio-data:/data --restart unless-stopped
  unset CMD_ARGS
  for _ in $(seq 1 24); do
    if curl -sf -o /dev/null "http://127.0.0.1:${S3_PORT}/minio/health/live"; then break; fi
    sleep 5
  done
  CAN_TEST_S3_ENDPOINT="http://127.0.0.1:${S3_PORT}" CAN_TEST_S3_REGION=us-east-1 \
    CAN_TEST_S3_BUCKET="${CAN_TEST_S3_BUCKET:-$S3_BUCKET_DEFAULT}" \
    CAN_TEST_S3_ACCESS_KEY="$(cat "$PROVISION_DIR/s3.access")" \
    CAN_TEST_S3_SECRET_KEY="$(cat "$PROVISION_DIR/s3.secret")" \
    bun "$(dirname "$0")/provision-s3-bucket.mjs"
  echo "s3 up: http://127.0.0.1:${S3_PORT}"
}

s3_exports() {
  echo "export CAN_TEST_S3_ENDPOINT=\"http://127.0.0.1:${S3_PORT}\""
  echo "export CAN_TEST_S3_REGION=\"us-east-1\""
  echo "export CAN_TEST_S3_BUCKET=\"${CAN_TEST_S3_BUCKET:-$S3_BUCKET_DEFAULT}\""
  echo "export CAN_TEST_S3_ACCESS_KEY=\"$(cat "$PROVISION_DIR/s3.access")\""
  echo "export CAN_TEST_S3_SECRET_KEY=\"$(cat "$PROVISION_DIR/s3.secret")\""
}

s3_down() { docker stop "$MINIO_CONTAINER" >/dev/null 2>&1 || true; echo "s3 stopped"; }

# browser_setup installs the pinned playwright client and the launchServer
# entry into the persisted /srv volume exactly once (marker-guarded).
browser_setup() {
  if docker exec "$FF_CONTAINER" test -f /srv/.can-ff-ready >/dev/null 2>&1; then return 0; fi
  docker exec "$FF_CONTAINER" sh -c 'mkdir -p /srv && cd /srv && npm init -y >/dev/null 2>&1 && npm i --save-exact playwright@1.55.1 >/dev/null 2>&1'
  printf '%s\n' "$FF_SERVER_JS" | docker exec -i "$FF_CONTAINER" sh -c 'cat > /srv/pw-server.mjs'
  docker exec "$FF_CONTAINER" sh -c 'node --check /srv/pw-server.mjs && touch /srv/.can-ff-ready'
}

browser_server_running() {
  docker exec "$FF_CONTAINER" sh -c 'ps -eo args 2>/dev/null | grep -q "[p]w-server.mjs"' >/dev/null 2>&1
}

browser_up() {
  need docker; need node
  CMD_ARGS='sleep infinity'
  container_up "$FF_CONTAINER" "$FF_IMAGE" "-p 127.0.0.1:${FF_PORT}:${FF_PORT}" \
    -v can-ff-srv:/srv --restart unless-stopped
  unset CMD_ARGS
  browser_setup
  # The container's main process is sleep; the ws server runs beside it
  # and needs an explicit (re)start after every fresh container start.
  if ! browser_server_running; then
    # shellcheck disable=SC2086
    docker exec -d "$FF_CONTAINER" sh -c "cd /srv && node pw-server.mjs $FF_PORT > /srv/server.log 2>&1"
    sleep 2
  fi
  for _ in $(seq 1 24); do
    if docker exec "$FF_CONTAINER" cat /srv/server.log 2>/dev/null | grep -q '^WS=ws://'; then break; fi
    if ! browser_server_running; then
      echo "firefox server died; log:" >&2
      docker exec "$FF_CONTAINER" cat /srv/server.log >&2 || true
      return 1
    fi
    sleep 5
  done
  docker exec "$FF_CONTAINER" cat /srv/server.log 2>/dev/null | grep -q '^WS=ws://' \
    || { echo "firefox server did not publish its endpoint" >&2; return 1; }
  node -e 'const s=require("node:net").connect(Number(process.argv[1]),"127.0.0.1",()=>{s.end();process.exit(0)});s.on("error",()=>process.exit(1));setTimeout(()=>process.exit(1),10000).unref()' "$FF_PORT" \
    || { echo "firefox ws port unreachable from the host" >&2; return 1; }
  echo "browser up: firefox ws 127.0.0.1:$FF_PORT (eval exports for CAN_FIREFOX_WS)"
}

browser_exports() {
  need docker
  ws="$(docker exec "$FF_CONTAINER" cat /srv/server.log 2>/dev/null | grep '^WS=' | tail -n 1 | sed 's/^WS=//; s#://[^:/]*:#://127.0.0.1:#')"
  if [ -z "$ws" ]; then echo "firefox runner not up (run: provision-local.sh browser up)" >&2; exit 1; fi
  echo "export CAN_FIREFOX_WS=\"$ws\""
  echo "# Optional host alias for firefox legs (default when unset): CAN_FIREFOX_HOST_ALIAS=host.docker.internal"
}

browser_down() { docker stop "$FF_CONTAINER" >/dev/null 2>&1 || true; echo "browser stopped"; }

cmd="${1:-}"; sub="${2:-}"; shift 2 2>/dev/null || { echo "usage: $0 {db|s3|browser} {up|down|exports|mkdb}" >&2; exit 1; }
case "$cmd/$sub" in
  db/up) db_up ;;
  db/exports) db_exports ;;
  db/mkdb) db_mkdb "$@" ;;
  db/down) db_down ;;
  s3/up) s3_up ;;
  s3/exports) s3_exports ;;
  s3/down) s3_down ;;
  browser/up) browser_up ;;
  browser/exports) browser_exports ;;
  browser/down) browser_down ;;
  *) echo "usage: $0 {db|s3|browser} {up|down|exports|mkdb}" >&2; exit 1 ;;
esac
