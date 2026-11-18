# Production database

*Operations notes. Lives at `docs/ops/production-database.md`. No secrets in this file.*

## Where it is

PostgreSQL 16 on the `safadev` droplet (SFO2, 1 GB memory), the same machine as the API. Installed from Ubuntu 24.04's packages on 2026-11-17.

- It listens on **localhost only**. Nothing on the internet can connect to it.
- Database `muster`, owned by the role `muster`. The API never connects as `postgres`.
- The API's connection string is `DATABASE_URL` in the service's environment file. Find the file with `systemctl cat muster` (look for `EnvironmentFile=`). It uses `localhost:5432` and `sslmode=disable`, which is fine for a connection that never leaves the machine.
- Pool: the API opens between 2 and 10 connections (`DB_MIN_CONNS`, `DB_MAX_CONNS`). Postgres allows 100.

## Why it is here

It was on Neon's free tier until 2026-11-17. The notification dispatcher queries the database every 30 seconds (`APP_DISPATCHER_INTERVAL`), so the database never went idle, Neon never suspended it, and the monthly compute allowance ran out. Neon then refused every connection. There were no users, so nothing was migrated; the schema was rebuilt from the goose migrations.

A fixed-price database has no usage meter, so the polling is harmless here.

## Migrations

`deploy.yml` runs them before shipping the binary:

1. Writes the deploy SSH key.
2. Opens a tunnel: the runner's `localhost:5433` to the droplet's `localhost:5432`.
3. Runs `goose up` against the secret `PROD_DATABASE_URL`, which points at `localhost:5433`.

The workflow triggers on changes under `backend/`. A change to the workflow file alone does not start it; use Actions → deploy → Run workflow.

## Reaching it from a laptop

Open a tunnel and leave it running:

```
ssh -N -L 5433:localhost:5432 <user>@<droplet>
```

Then connect any client (`psql`, DataGrip, goose) to `localhost:5433` with the `muster` role. Port 5433 avoids the local dev database on 5432.

On the droplet itself, the admin shell is `sudo -u postgres psql`.

## Backups

A cron job in the `postgres` user's crontab, daily at 09:15 UTC:

```
15 9 * * * pg_dump -Fc muster > /var/backups/muster/muster-$(date +\%F).dump && find /var/backups/muster -name '*.dump' -mtime +14 -delete
```

- Dumps are in `/var/backups/muster`, one per day, 14 days kept.
- Check it: `ls -lh /var/backups/muster` should show a file dated today.
- Edit it: `sudo crontab -u postgres -e`.

### Restore

Into a new, empty database, so a bad restore can't damage the live one:

```
sudo -u postgres createdb muster_restore -O muster
sudo -u postgres pg_restore -d muster_restore /var/backups/muster/muster-YYYY-MM-DD.dump
```

Inspect it, then either copy out what is needed or switch `DATABASE_URL` to it.

**Not yet done:** this restore has never been tried, and the dumps are on the same disk as the database. Before real users: copy the dumps off the droplet and run one restore end to end.

## Watching it

```
free -h
```

Read "available", not "free". On 2026-11-17, with Postgres running and the API connected: 432 MB used, 529 MB available, 41 MB of swap in use. If "available" falls below about 100 MB, or swap use reaches the hundreds of megabytes, the droplet is too small for everything on it.

## If it outgrows the droplet

DigitalOcean's managed PostgreSQL is the next step: automatic backups and its own machine, for a monthly charge. The move is a `pg_dump` from here, a restore there, and a new `DATABASE_URL` and `PROD_DATABASE_URL` (the tunnel step in `deploy.yml` goes away if the managed database accepts connections from the runner).
