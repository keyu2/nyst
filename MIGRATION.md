# NYST migration acceptance — 2026-11-03

Source: `/Users/aki/Documents/subway` (retained).
Destination: `/Users/aki/Documents/nyst`.

Status: subway-product implementation migration and the recorded parity acceptance are complete. This is a platform migration, not a redesign. Retired LIRR and Metro-North remain excluded. PostgreSQL was optional in the requested architecture and is not introduced; neither is Redis. Deployment/publishing was not requested or performed.

## Architecture and commands

- SvelteKit 3 + Svelte 5 + TypeScript under `web/src/routes/` and `web/src/lib/`.
- Go backend under `server/`; production embeds `web/dist` including `_app` resources in one executable.
- Original MTA static GTFS, realtime feeds, alert feed, metadata source and public assets preserved.
- REST snapshots and one shared SSE connection per browser tab, with polling fallback. Bounded snapshot caching replaces data rather than retaining feed history.
- Development ports are now paired: `NYST_PORT=x npm run dev` starts the webpage on x and Go on x+1 (default 5173/5174), with strict failure for occupied/invalid ports. `NYST_API_PORT` is an optional explicit override. Verified a custom pair and its API proxy, plus invalid and occupied backend-port rejection.
- `npm run dev` from the root starts the existing combined Vite/Go watcher. No Air or concurrently dependency. `make dev` is an equivalent entry point. Frontend dependencies are installed inside `web/`.
- `npm run check`, `npm run build`, `npm start`. Development saves update automatically; production must be rebuilt.
- The two preexisting development servers on 5173/5175 and original Next.js server on 3000 are not to be stopped without checking ownership. Test-only production port 3038 and the dedicated Chromium test browser were stopped after acceptance.

## Settings update after migration

User-selected labels now use Theme, Show seconds (Off/On), Preview, Arrival estimate, Home page (Traditional/Slider), Map, Highlight arriving trains and Line layout on phones. Station view is labeled Station page (Arrival list/Station board). Source and destination are synchronized. The mobile layout defaults to rows for missing/invalid settings, while an explicitly stored grouped choice is preserved.

## Cross-borough strip fix

Both projects now color the borough-boundary connector from the same `trackAbove` value as the adjoining rail, instead of always using the route color. Browser checks at 390px and 1280px verify matching color, width and alignment in inactive, normal, express-skipping and route-wide no-service cases. Source checks/build and destination checks/build pass.

## Station route groups and active-trip markers

Both projects group related station route tabs in one rounded outline while retaining individual family selection, including local/express bullets. Station maps filter lines and vehicles to the selected trunk group without recreating the map or changing realtime sources. Map markers share a departure check across global, station and trip maps: future starts, delayed origin departures and canceled trips are hidden. Running trips remain visible when their next stop is in the future. Branch track graphs exclude branch stops from the main spine so a representative express pattern cannot shortcut traversed local track; skipped stop dots and names remain inactive.

Source checks, 98 tests and build pass. Destination checks, 52 frontend tests, Go tests and build pass. Browser checks at 390px and 1280px verify grouped tabs, individual selection, actual map line/vehicle filtering, no horizontal overflow and green express branch rails through gray skipped stops.

## Shared map rendering

Both projects now use sustained corridor matching, per-service lanes on shared center paths, rounded screen-space bends, and independently positioned stop dots for each active service. Existing active-service metadata determines the dots; station names are deduplicated and adjacent dots are connected. Vehicles and direction arrows snap to the same rendered service geometry. Source data URLs, feed processing and original bullet SVGs are unchanged.

Camera changes redraw buffered viewport geometry through a single animation frame, with spatial indexes and a bounded single-layout cache. Browser checks cover Midtown, Queens Boulevard, Brooklyn, zoom, rotation, pitch, both themes and a 390px mobile viewport. Independently measured station and vehicle alignment stayed within 0.02 screen pixels in the tested views. Station-detail and trip navigation passed with no runtime exceptions. Source: 107 tests and build pass. Destination: 61 frontend tests, Go tests and build pass.

## Accepted behavior

Home, Favorites, Nearby, Status, Settings, route/branch strips, station arrivals/direction/terminus controls, trip timelines, route alerts, global/station/trip maps, vehicle and station navigation, Display Board, session filters, fullscreen fitting, preferences and favorite persistence are migrated. Original URLs and Back/Home behavior are retained. Realtime remains authoritative; successful no-service results are distinct from unavailable feeds.

- Both layouts share preferences between Home and Status.
- Countdown seconds, minutes/stops progress, board-only inversion setting, map enable setting and theme settings work.
- Board inversion starts at 59 seconds; 60 seconds remains normal. Paired alerts rotate every 15 seconds and loop without resetting on identical realtime snapshots. A single page has full progress.
- Previous/intermediate trip stops expand, transfer links navigate, saved stations survive missing discovery results, favorite updates synchronize immediately.
- Maps use the original service-lane algorithms, original SVG vehicles/bullets, station labels and click navigation. Sources update without reconstructing the viewport; theme changes retain transit layers.
- Optional branches follow future realtime service, including the common Rockaway stem. Express traversed track and actual stopping service are separate; skipped dots/names remain gray.

## Same-input source versus Go acceptance

A single real MTA snapshot was captured with a frozen clock. Both implementations consumed identical upstream bytes. The source itself was not modified for this capture.

- All 29 route responses and service availability match.
- All 443 station complexes, service IDs and their ordering match.
- All 13 selected station responses (852 arrivals) match after schema adaptation.
- All 24 selected trip paths, times, origins, termini and marker positions match.
- Alerts/statuses, four station maps and the global map match.
- Go-only serialization defaults and source-only rendering/debug fields are normalized; bearing tolerance is 0.000001 degrees and coordinate tolerance 0.00000001 degrees. These do not change visible data behavior.
- Shared JavaScript versus TypeScript geometry functions were compared directly on the full input. Service lanes, rendered geometry, vehicles and stations match.
- Full end-to-end Go-input versus source-input rendering also matches: all 443 global station placements have zero coordinate difference; 893 global vehicles differ by at most 1.59e-14 degrees. The four station-map placement comparisons pass at the same precision.

Evidence outside the application: `/tmp/nyst-parity-20261103/` contains upstream bytes, manifest, source/go JSON and differences. `/tmp/nyst-capture-parity.mjs` and `/tmp/nyst-diff-parity.py` were the audit tools. The temporary Go snapshot-export test was removed after comparison; focused durable regressions remain in `server/mta/*_test.go`.

## Corrections established by comparison

- Preserve 6X/7X/FX through discovery and alert distribution; collapse only at the original presentation boundaries.
- Read public route IDs from routes.txt, preserve numeric station-name order and stable original order for ties.
- Equal-time arrivals and equal-score static trip matches retain feed order.
- Match the original GTFS versus metadata accessibility sources for raw stop objects and page headings.
- Preserve route-family map markers and original snapping bearings, with frontend station-route schema adaptation.
- Guard off-route terminal processing (E/W missing-route panic), departure-only timing and protobuf delay defaults.
- Preserve the noon-offset service-day rule across DST. Go embeds timezone data and reuses the New York location so minimal Linux hosts do not silently fall back to UTC.
- Preserve prior snapshot fallback, partial-feed behavior, duplicate-feed preference, cancellation handling and service calendar rules.
- Restore exact original icon paths, alert classes, favorites fallback/notification order and carousel dependency semantics.
- Apply stored theme before first paint. Use raw immutable map snapshots to avoid deeply proxied geometry processing.

## Browser and build acceptance

Actual Chromium checks compared the running original and migrated applications at 390px and 1280px:

- Route station counts, station tabs, typography/background, Back/Home, overflow and focused E-to-N navigation match.
- Actual 637-to-D21 transfer links and the same real E trip's 20 stops, next stop, footer and collapse controls match.
- Fullscreen board entry/exit and fitting match after layout settles: 800px viewport, 895px content, scale 0.893855, eight trains and one alert in the compared snapshot.
- Separate deterministic browser fixtures cover the 59/60-second boundary, paired 15-second cycling despite frequent refresh, single-page progress, seconds/inversion toggles, timeline expansion, progress preferences, favorites and Home/Status/settings synchronization. No browser exceptions in the accepted runs.
- Dark/light maps were visually checked using software WebGL. The first headless failure was traced to unavailable WebGL2; software rendering verified application behavior.
- All 35 original public files hash-identical, including bullet SVGs/icons/fonts/worker/favicon. Six stylesheets match after whitespace/comment normalization.
- `npm run check`: zero Svelte errors/warnings, 50 frontend tests, all Go tests pass.
- `npm run build`: static frontend and embedded Go executable build successfully.
- Standalone production smoke: deep page URLs, original SVG, map worker, immutable app chunks and actual E alert API return 200. Test service shuts down cleanly.

Browser evidence: `/tmp/nyst-live-ui-comparison.json`, `/tmp/source-station-390.png`, `/tmp/nyst-station-390.png`, `/tmp/source-trip-final.png`, `/tmp/nyst-trip-final.png`, and the fullscreen/board fixture screenshots under `/tmp/`.

## Verification limits

Physical iOS Safari hardware and a prolonged production soak were not exercised. Browser mobile emulation is not a device test. Original map alignment heuristics are preserved rather than replaced with a new track model; this migration does not certify the original map as track-survey accurate. Do not promise absolute freedom from all future feed edge cases or uninterrupted external MTA availability.
