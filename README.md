# ShopKaro

**Tap-to-sell billing for Indian kirana shops — works even when the internet doesn't.**

Entry for the [DEV Hacktoberfest 2026 Challenges](https://dev.to/challenges/hf26), Week 2: *Side Hustle* (Oct 12–18, 2026).

| Mobile (360px) | Desktop (1280px) |
| --- | --- |
| ![Sell screen on a phone](docs/screenshots/sell-mobile.png) | ![Sell screen on desktop](docs/screenshots/sell-desktop.png) |

| Stats | Reorder list | Weekly report |
| --- | --- | --- |
| ![Stats dashboard](docs/screenshots/stats-desktop.png) | ![Reorder forecast](docs/screenshots/reorder-mobile.png) | ![Weekly report](docs/screenshots/report-desktop.png) |

## Why

A kirana shopkeeper serves 150+ customers a day. Billing apps today assume a stable
internet connection, English literacy, and a keyboard. When the network drops mid-sale
(and in a small shop it does), the queue stalls and the customer waits.

ShopKaro is built for that reality:

- **Tap tiles, not forms.** Products are big tappable tiles with monogram chips — no typing
  during a sale. `×1 ×2 ×3 ×5` multipliers cover the common cases in one tap.
- **Works offline.** Every sale is written to IndexedDB first, then replayed to the API with
  idempotent `client_id`s when the connection returns. The UI shows exactly how many sales are
  waiting to sync.
- **Speak your language.** "2 Maggi, 1 doodh, do parle g" is parsed into structured sales —
  Hinglish aliases, Hindi numerals (`do` = 2), and fuzzy matching included.
- **Answers in plain language.** A bilingual (English/हिन्दी) weekly report tells the shopkeeper
  what sold, what didn't, and what to order next.

## Feature tour

| Screen | What it does |
| --- | --- |
| **Sell** | Searchable product grid, category chips, quantity multipliers, recent sales with +/- edit and undo, AI voice/text entry (FAB). |
| **Products** | Add/edit products, comma-separated voice aliases, optional pack photo that Gemma reads to fill in name/price/category. |
| **Stats** | Revenue, bills, average bill, busiest hour; revenue-by-day bars, best sellers, slow movers, sales-by-hour, category breakdown (7/28 day ranges). |
| **Reorder** | Next week's forecast per product — last week → forecast → order quantity with confidence levels. Saves as a report job. |
| **Report** | Plain-language weekly summary in English or Hindi, the numbers behind it, and the status of both scheduled jobs. |

The layout is mobile-first and verified at **360px, 768px and 1280px** — a sidebar on desktop,
a bottom tab bar on phones.

## Quick start

```bash
git clone https://github.com/AarishMansur/ShopKaro.git
cd ShopKaro
docker compose up
```

Open **http://localhost:3000**. On first boot the API seeds a demo shop (44 real kirana
products, ~10 weeks of sales history) — idempotent, so restarts never duplicate data.

Useful endpoints while it runs:

- Web app: `http://localhost:3000`
- API: `http://localhost:8080/api/health`
- Postgres/Timescale: `localhost:15432` (`shopkaro` / `shopkaro`)

## Deploy on Render

The repo ships a [render.yaml](render.yaml) blueprint:

1. Fork this repository.
2. In Render: **New + → Blueprint** → point it at your fork.
3. Render provisions Postgres, builds the Go API and SvelteKit web images, and wires
   `API_ORIGIN` between them. The API seeds the demo shop on first boot.
4. Open the `shopkaro-web` URL and start tapping sales.

## Stack

| Layer | Tech |
| --- | --- |
| Frontend | SvelteKit 5 (runes) + TypeScript, adapter-node, PWA service worker, IndexedDB offline queue |
| Backend | Go 1.27, net/http, pgx |
| Database | PostgreSQL + **TimescaleDB** (`sales` is a hypertable on `(id, sold_at)`) |
| Scheduling | Built-in scheduler, or **Temporal** durable cron schedules (`--profile temporal`) |
| Forecast | **TabPFN** (Prior Labs REST API) with a seasonal-trend fallback |
| Voice | **Whisper** sidecar (faster-whisper) with browser SpeechRecognition fallback |
| LLM | **Gemma 3** via **Ollama** — text parsing, weekly summaries, pack-photo reading |
| Deploy | Docker Compose, Render blueprint |

## Optional profiles

Everything works without them — each integration degrades gracefully to a built-in fallback.

```bash
# Gemma (Ollama) + Whisper voice sidecar — pulls gemma3:4b on first run
docker compose --profile ai up

# Durable Temporal schedules for the weekly summary + reorder report,
# with the Temporal UI on http://localhost:8088
TEMPORAL_ENABLED=true docker compose --profile temporal up

# TabPFN forecast: put your key in .env (see .env.example)
TABPFN_API_KEY=... docker compose up
```

### Tiger Data (hosted Timescale)

Set `TIGER_DB=` in `.env` to a [Tiger Data](https://www.tigerdata.com/) connection string and
the stack stores sales in the cloud instead of the local `db` container. The hypertable
migration still runs automatically on boot.

## Voice entry

Tap the microphone FAB on the Sell screen:

1. **Whisper mode** (profile `ai` + `WHISPER_URL` set): records in the browser and
   transcribes with the open-source Whisper model.
2. **Browser mode**: uses the Web Speech API (`hi-IN`) — no server needed.
3. **Type mode**: `2 Maggi, 1 doodh` parses the same way in either case.

The parsed entries are shown for review before anything is recorded.

## Offline behaviour

- Sales are queued in IndexedDB **before** any network call.
- Sync replays the queue as a batch; the API deduplicates on `client_id`, so a retried
  batch never double-charges (verified by replaying the same batch twice).
- The sidebar/badge shows `Online`/`Offline` and how many sales are waiting.
- The app shell (HTML, CSS, JS, icons) is cached by a service worker, so the Sell screen
  opens without a network.

## API overview

| Method & path | Purpose |
| --- | --- |
| `GET /api/health`, `GET /api/features` | Health, integration availability |
| `GET/POST /api/products`, `PUT/DELETE /api/products/{id}` | Catalogue |
| `POST /api/products/from-photo` | Gemma vision → product suggestion |
| `GET/POST /api/sales/batch` | Recent sales, idempotent batch insert |
| `PATCH/DELETE /api/sales/{id}` | Adjust quantity, undo |
| `GET /api/dashboard/*` | Summary, best sellers, slow movers, by-day, by-hour |
| `GET /api/reorder/forecast` | Next-week reorder quantities |
| `POST /api/ai/parse`, `POST /api/ai/voice` | Hinglish parse, Whisper transcription |
| `GET /api/ai/summary` | Bilingual weekly report |
| `GET /api/jobs/status`, `POST /api/jobs/run` | Scheduled job status, manual trigger |

The browser only ever talks to same-origin `/api`; the SvelteKit server proxies to the Go
API (`API_ORIGIN`), so there is one URL in dev, Docker and Render.

## Local development

```bash
# Terminal 1 — database (any Postgres works; Timescale enables the hypertable)
docker run -d --name shopkaro-db -e POSTGRES_USER=shopkaro \
  -e POSTGRES_PASSWORD=shopkaro -e POSTGRES_DB=shopkaro \
  -p 15432:5432 timescale/timescaledb:latest-pg16

# Terminal 2 — Go API (auto-migrates, then seeds)
cd backend
DATABASE_URL=postgres://shopkaro:shopkaro@localhost:15432/shopkaro?sslmode=disable go run ./cmd/api

# Terminal 3 — SvelteKit
cd frontend
npm install
API_ORIGIN=http://localhost:8080 npm run dev
```

Quality gates:

```bash
cd backend  && go vet ./... && go test ./...
cd frontend && npm run check && npm run build
```

## Repository layout

```
backend/           Go API: store, AI parsing, forecast, jobs (cmd/api, cmd/seed)
frontend/          SvelteKit app: routes, lib (api, offline queue, sync), components
services/whisper/  Open-source Whisper transcription sidecar
docker-compose.yml Stack + optional ai/temporal profiles
render.yaml        Render blueprint (api, web, Postgres)
.env.example       Every configuration knob, documented
docs/screenshots/  Responsive screenshots used above
```

## Partner prize categories entered

This entry is submitted to the following
[partner tracks](https://dev.to/challenges/hf26):

- **Gemma** — Gemma 3 via Ollama for Hinglish sale parsing, bilingual weekly summaries,
  and pack-photo product entry.
- **TabPFN** — TabPFN REST API powers the next-week reorder forecast (seasonal fallback when
  no key is set).
- **Temporal** — durable cron schedules for the Monday morning summary and Sunday evening
  reorder report, with retries and the Temporal UI.
- **Tiger Data** — sales live on TimescaleDB hypertables (set `TIGER_DB` for hosted Tiger Data).
- **Render** — one-click deploy via the included `render.yaml` blueprint.

## Status

Built for DEV Hacktoberfest 2026 Week 2 (deadline Oct 18, 2026). Tested end-to-end with
`docker compose up`: seeded catalogue, tap-to-sale through the proxy, idempotent offline
replay, Timescale hypertable, Temporal schedules, and responsive layouts at 360/768/1280px.
