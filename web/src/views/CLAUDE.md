# web/src/views

One file per UI pane, each a `render<Name>(container, ...args)` function. Composed together in `main.ts`.

- `browser.ts` — `renderBrowser`: server-side directory browser + video multi-select (checkboxes, disabled for non-video files per `entry.isVideo` from the backend). Persists the last-visited path in `localStorage` (`whisper-gui:lastPath`) so it reopens where the user left off.
- `jobForm.ts` — `renderJobForm`: model/language `<select>`s plus the "Start Transcription" button. Purely presentational — `onStart(model, language)` hands control back to `main.ts`, which does the actual `createJob` calls per selected file.
- `jobList.ts` — `renderJobList`: polls-on-demand (`refresh()`) list of all jobs, newest first (matches the backend's `ListJobs` ordering). Click a row → `onSelect(id)`.
- `jobDetail.ts` — `renderJobDetail`: the single most stateful view. Loads the job once via `getJob`, then opens an SSE stream (`openJobStream`) for live `log`/`progress`/`state` updates. Handles the sidecar-copy note (saved next to video vs. copy failed — see `jobmanager.copySrtNextToVideo`), the "Save to Video Folder"/"Save to Folder…" buttons (`moveSrt` in `../api.ts`, folder picker from `folderPicker.ts`), and the cancel button (hidden once the job reaches a terminal state). Always call `.destroy()` on the previous instance before rendering a new job's detail (see `main.ts`'s `openJobDetail`) — it closes the SSE connection.
- `folderPicker.ts` — `openFolderPicker(onSelect)`: a directory-only modal variant of `browser.ts`'s browser, used by `jobDetail.ts` to let the user pick an arbitrary destination folder for the `.srt`. Remembers the last folder browsed to in `localStorage` (`whisper-gui:lastSrtDestDir`), same pattern as `browser.ts`'s own last-visited path, so it reopens where the user left off.

## Conventions

- State labels (`queued`/`running`/`done`/`failed`/`cancelled` → display text) are duplicated as `STATE_LABEL` in both `jobList.ts` and `jobDetail.ts` rather than shared — keep both in sync if the state set ever changes.
- These views talk to the backend only through `../api.ts` and `../sse.ts` — no direct `fetch`/`EventSource` calls here.
