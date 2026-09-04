# web/src

Vanilla TypeScript app, no framework. DOM is built directly, not templated.

- `dom.ts` — the entire "framework": `el(tag, props, children)` creates an element and wires up attributes/`class`/`onXxx` event listeners in one call; `clear(node)` empties it. Every view in `views/` is built with just this.
- `api.ts` — typed `fetch` wrappers for every backend endpoint (`/api/browse`, `/api/models`, `/api/settings`, `/api/jobs*`). `request<T>()` centralizes error handling: a non-OK response throws using the server's JSON `{error}` body when present, falling back to the HTTP status text.
- `sse.ts` — `openJobStream(jobId, handlers)`: wraps `EventSource` for a job's `log`/`progress`/`state` events. Relies on the browser's native auto-reconnect (never manually disabled) — because the server replays its whole capped log buffer from scratch on every reconnect, a per-job monotonic `seq` number is tracked here to drop already-rendered lines instead of re-appending duplicates.
- `sortControls.ts` — shared, framework-free sort utilities used by both `views/browser.ts` and `views/folderPicker.ts`: `sortEntries()` (pure comparator over `BrowseEntry[]`, folders always pinned above files regardless of direction), `parseSortState()` (validates a `"key:dir"` string from `localStorage` against a caller-supplied list of allowed keys, falling back to `{key:'name',dir:'asc'}`), and `renderSortRow()` (renders a row of clickable "column header" buttons — click to sort ascending, click again to reverse). Not itself a `views/` pane; a plain utility module like `dom.ts`/`api.ts`.
- `types.ts` — all shared TS interfaces (`Job`, `BrowseEntry`, `ModelInfo`, `LogEvent`, etc.), hand-kept in sync with the Go JSON structs in `internal/server`/`internal/jobmanager`. There is no codegen — when a backend field changes, update this file too.
- `main.ts` — top-level composition: lays out the four panes (`browser`, `form`, `list`, `detail`), wires selection → job form → job creation → job list refresh → job detail. This is the only place that owns cross-view state (`selectedPaths`). Before creating jobs, it also checks `browser.getEntry(path).hasSrt` for each selected file and shows a native `confirm()` listing any that already have a subtitle, since job creation always overwrites.
- `style.css` — all styling; no CSS framework, no CSS modules/scoping (plain global class names, matched against what `views/*.ts` render).
- `views/` — one file per UI pane; see its own `CLAUDE.md`.

## Conventions

- Every view module exports a `render<Name>(container, ...)` function returning a small controller object (e.g. `{ refresh() }`, `{ destroy() }`) — follow this shape for new views rather than returning DOM nodes directly.
- No state management library — state lives in closures inside each view, with cross-view coordination done explicitly in `main.ts`.
