# NYST

The SvelteKit, TypeScript and Go implementation of New York Subway Tracker. It contains the subway product only. The retired LIRR and Metro-North code is not included.

## Architecture

- `web/src/routes/` contains SvelteKit pages.
- `web/src/lib/` contains shared TypeScript logic and Svelte components.
- `web/public/` preserves the original bullets, fonts, icons and map worker.
- `server/` contains the Go API, MTA clients, and bounded live-data cache.
- `main.go` embeds the compiled interface and starts one production server.

Static GTFS supplies station topology, scheduled service, and fallback data.
Realtime MTA feeds remain authoritative whenever they are available. A successful response with no upcoming service does not fall back to the timetable. Static topology can still describe skipped stations without marking them as served.

REST endpoints deliver snapshots. One SSE connection per browser tab announces fresh realtime data; REST polling resumes if that connection is unavailable.

The live cache replaces complete snapshots instead of accumulating feed history. If an MTA refresh fails, the API continues serving the most recent valid snapshot.

## Development

Install the interface dependencies once:

```bash
cd web
pnpm install
cd ..
```

Start development from the project directory:

```bash
npm run dev
```

The default ports are 5173 for the webpage and 5174 for Go. To choose a pair, set the webpage port; Go defaults to the next port:

```bash
NYST_PORT=3000 npm run dev
```

This uses 3000 for the webpage and 3001 for Go. Occupied ports cause startup to fail instead of silently changing the pair. `NYST_API_PORT` remains available as an explicit backend override.

Open the address printed by Vite, normally <http://localhost:5173>. Svelte and CSS edits update automatically. Go edits rebuild and restart the backend automatically, then reload the page. Stop both with Ctrl+C; the terminal session can stay open.

`make dev` also starts the same development runner. You can run `pnpm dev` from `web/` as well. No Air or concurrently installation is needed.

## Production

```bash
make build
./nyst
```

Open <http://localhost:3000>. Production uses the compiled interface and does not watch source files.

## Verification

```bash
cd web && pnpm check && pnpm test && pnpm build
cd .. && go test ./... && go build -o nyst .
```
