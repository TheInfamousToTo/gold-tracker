#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== Gold Tracker Database Setup ==="
read -p "PostgreSQL host [localhost]: " PGHOST
PGHOST=${PGHOST:-localhost}

read -p "PostgreSQL port [5432]: " PGPORT
PGPORT=${PGPORT:-5432}

read -p "Postgres superuser [postgres]: " PGSUPERUSER
PGSUPERUSER=${PGSUPERUSER:-postgres}

read -p "New DB name [gold_tracker]: " DBNAME
DBNAME=${DBNAME:-gold_tracker}

read -p "New app username [gold_admin]: " DBUSER
DBUSER=${DBUSER:-gold_admin}

read -sp "Enter password for superuser ($PGSUPERUSER): " PGSUPERPASS
echo
read -sp "Enter password for app user ($DBUSER): " DBUSERPASS
echo
echo "---"

# Create database + role using the superuser connection
export PGPASSWORD="$PGSUPERPASS"

echo "Creating database and role..."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGSUPERUSER" -d postgres -c "CREATE DATABASE $DBNAME;" 2>/dev/null || echo "  - Database already exists, skipping."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGSUPERUSER" -d postgres -c "CREATE USER $DBUSER WITH ENCRYPTED PASSWORD '$DBUSERPASS';" 2>/dev/null || echo "  - Role already exists, updating password."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGSUPERUSER" -d postgres -c "ALTER USER $DBUSER WITH PASSWORD '$DBUSERPASS';"
psql -h "$PGHOST" -p "$PGPORT" -U "$PGSUPERUSER" -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DBNAME TO $DBUSER;"

echo "Granting schema privileges inside '$DBNAME'..."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGSUPERUSER" -d "$DBNAME" -c "GRANT ALL ON SCHEMA public TO $DBUSER;"

# Build the schema as the app user
export PGPASSWORD="$DBUSERPASS"

# The whole schema lives in backend/migrations, which the API also
# applies at every boot — running the same files here is what keeps a
# fresh install and an upgraded one identical. Each file is idempotent,
# so this is safe to re-run.
echo "Creating tables and view..."
for migration in "$SCRIPT_DIR"/backend/migrations/*.sql; do
    echo "  - $(basename "$migration")"
    psql -h "$PGHOST" -p "$PGPORT" -U "$DBUSER" -d "$DBNAME" -q -v ON_ERROR_STOP=1 -f "$migration"
done

unset PGPASSWORD

echo "=== Done. Database '$DBNAME' is ready for user '$DBUSER'. ==="
