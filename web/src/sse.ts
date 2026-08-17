import type { LogEvent, ProgressEvent, StateEvent } from './types'

export interface JobStreamHandlers {
  onLog?: (e: LogEvent) => void
  onProgress?: (e: ProgressEvent) => void
  onState?: (e: StateEvent) => void
}

const TERMINAL_STATES = new Set(['done', 'failed', 'cancelled'])

/**
 * Opens an SSE connection for a job. Relies on EventSource's native
 * auto-reconnect (never disabled) -- the server replays its capped log
 * buffer from scratch on every reconnect, so a per-job monotonic sequence
 * number is used here to skip anything already rendered, rather than
 * re-appending the whole replay on top of existing DOM content.
 */
export function openJobStream(jobId: string, handlers: JobStreamHandlers): () => void {
  const es = new EventSource(`/api/jobs/${encodeURIComponent(jobId)}/events`)
  let maxSeqSeen = -1

  es.addEventListener('log', (ev) => {
    const data: LogEvent = JSON.parse((ev as MessageEvent).data)
    if (data.seq <= maxSeqSeen) return
    maxSeqSeen = data.seq
    handlers.onLog?.(data)
  })

  es.addEventListener('progress', (ev) => {
    const data: ProgressEvent = JSON.parse((ev as MessageEvent).data)
    handlers.onProgress?.(data)
  })

  es.addEventListener('state', (ev) => {
    const data: StateEvent = JSON.parse((ev as MessageEvent).data)
    handlers.onState?.(data)
    if (TERMINAL_STATES.has(data.state)) {
      es.close()
    }
  })

  return () => es.close()
}
