# NetWatch

A lightweight, self-hosted Internet monitor for a home server. It continuously
checks your connection, tracks speed tests, computes uptime/availability
analytics, sends Telegram alerts when your Internet goes down or comes back,
and serves a live dashboard.

```
Connectivity checks ──▶ bbolt ──▶ Analytics ──▶ REST API ──▶ Dashboard
      (every 15s)                    │                          │
                                      └──▶ Telegram alerts ◀─────┘
Speed tests (every 6h) ───────────────────────▲
```

## Features

- **Connectivity monitoring** every 15s: DNS resolution, an HTTPS request,
  and a TCP reachability check (used as a ping substitute, so the service
  never needs raw-socket/root privileges).
- **Outage detection**: N consecutive failures (configurable) open an
  outage record; recovery closes it and records the duration.
- **Speed tests** every 6h (or on demand) via the public speedtest.net
  network, behind a `Provider` interface so the backend can be swapped later.
- **Analytics** maintained incrementally (daily/monthly summary buckets) —
  no full-table scans on every request.
- **Telegram alerts**: Internet Down, Internet Restored, Daily Summary,
  Weekly Summary, and a Test button — all runtime-configurable via the API.
- **Retention**: raw connectivity checks are pruned after N days (default
  90); daily/monthly summaries are kept forever.
- **Dashboard**: Next.js + Tailwind + Recharts, dark mode, live-polling.

## Project layout

```
cmd/server/            entry point, dependency wiring, graceful shutdown
internal/
  api/                 Gin routes & handlers
  analytics/           incremental daily/monthly stats engine
  config/              Viper-based config.yaml + env loading
  database/            bbolt open/bucket management + generic JSON helpers
  middleware/           logging, recovery, CORS
  models/              shared domain types
  monitor/             connectivity checker + outage detection
  repository/          bbolt-backed persistence, one file per domain
  scheduler/           cron/ticker orchestration of all background jobs
  services/
    retention/         raw-data cleanup
    speedtest/         Provider abstraction + Ookla implementation
  telegram/            Bot API client, message templates, notifier service
  utils/               logger, id generation
configs/config.yaml    default configuration
web/                   Next.js dashboard
docker/                Dockerfiles for backend & frontend
```

## Running locally

**Backend** (Go 1.24+):

```bash
make build
make run            # or: ./bin/netwatch-server --config ./configs
```

The API listens on `:8080` by default. Try `curl localhost:8080/api/status`.

**Frontend** (Node 20+):

```bash
cd web
cp .env.local.example .env.local   # set NEXT_PUBLIC_API_BASE_URL if needed
npm install
npm run dev
```

Visit `http://localhost:3000`.

## Running with Docker Compose

```bash
cp .env.example .env
# edit .env: set NEXT_PUBLIC_API_BASE_URL to how your BROWSER reaches the
# backend (your server's LAN IP/hostname — not "backend", which only
# resolves container-to-container), and your Telegram credentials.
docker compose up -d --build
```

- Backend: `http://<server>:8080`
- Dashboard: `http://<server>:3000`

Connectivity/speed test data persists in the `netwatch-data` volume.
`configs/config.yaml` is mounted read-only, so config changes just need a
container restart (`docker compose restart backend`) — no rebuild.

> `NEXT_PUBLIC_API_BASE_URL` is inlined into the frontend at **build** time
> (it runs in the browser, not the container), so changing it requires
> `docker compose up -d --build frontend`.

## Makefile targets

`make build` / `run` / `test` / `lint` (backend); `make web-install` /
`web-dev` / `web-build` / `web-lint` (frontend); `make docker-up` /
`docker-down` / `docker-logs` (compose); `make clean`.

## Configuration

Edit `configs/config.yaml` or override via `NETWATCH_*` environment
variables (e.g. `NETWATCH_MONITOR_INTERVAL_SECONDS=30`,
`NETWATCH_SERVER_PORT=9090`). See the file for every available key:
server, database, telegram, monitor, speedtest, retention, logging.

Some settings — Telegram credentials, monitor/speed-test interval, and
retention days — are also runtime-configurable via `GET`/`POST
/api/settings` and take effect without a restart (the scheduler re-reads
them each tick).

## REST API

| Method | Path | Description |
|---|---|---|
| GET | `/api/status` | Live status snapshot |
| GET | `/api/connectivity` | Raw checks (`?from=&to=&limit=`, RFC3339) |
| GET | `/api/speed/latest` | Most recent speed test |
| GET | `/api/speed/history` | Speed test history (`?from=&to=&limit=`) |
| POST | `/api/speedtest` | Trigger a speed test now |
| GET | `/api/downtime` | Recent outages (`?limit=`) |
| GET | `/api/analytics/daily` | `?date=YYYY-MM-DD` or `?from=&to=` range |
| GET | `/api/analytics/monthly` | `?month=YYYY-MM` or `?from=&to=` range |
| GET | `/api/settings` | Current settings |
| POST | `/api/settings` | Update settings |
| POST | `/api/telegram/test` | Send a test Telegram notification |

## Telegram setup

1. Create a bot with [@BotFather](https://t.me/BotFather) and copy the token.
2. Message the bot, then fetch your chat ID (e.g. via
   `https://api.telegram.org/bot<token>/getUpdates`).
3. Set `telegram_enabled: true`, `bot_token`, and `chat_id` — via
   `configs/config.yaml`, `POST /api/settings`, or the dashboard's Settings
   panel — then hit "Send test message".

## Security note

There is no authentication on the REST API — it's designed to run on a
trusted home LAN. If you expose it beyond your LAN, put it behind a reverse
proxy with auth (e.g. Caddy/Traefik + basic auth, or a VPN).

## Extending

The repository/service-interface split means new monitors (CPU, memory,
disk, Docker containers, SSL certs, website uptime, public IP changes, DNS,
UPS/battery, network devices) can be added as new `internal/<domain>`
packages with their own bbolt bucket and repository, wired into the
scheduler and API the same way `monitor` and `speedtest` are.
