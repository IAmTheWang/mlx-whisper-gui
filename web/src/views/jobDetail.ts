import type { Job, LogEvent, ProgressEvent, StateEvent } from '../types'
import { cancelJob, getJob, moveSrt, srtDownloadUrl } from '../api'
import { openJobStream } from '../sse'
import { openFolderPicker } from './folderPicker'
import { el } from '../dom'

export interface JobDetailController {
  destroy(): void
}

const STATE_LABEL: Record<string, string> = {
  queued: 'Queued',
  running: 'Running',
  done: 'Done',
  failed: 'Failed',
  cancelled: 'Cancelled',
}

const TERMINAL_STATES = new Set(['done', 'failed', 'cancelled'])

export function renderJobDetail(container: HTMLElement, jobId: string, onTerminal?: () => void): JobDetailController {
  const stateLabel = el('span', { class: 'detail-state' }, ['Loading…'])
  const progressFill = el('div', { class: 'progress-bar-fill' })
  const progressBar = el('div', { class: 'progress-bar' }, [progressFill])
  const progressText = el('span', { class: 'progress-text' }, [''])
  const cancelBtn = el('button', { class: 'cancel-btn' }, ['Cancel'])
  const downloadLink = el('a', { class: 'download-link' }, ['Download .srt'])
  downloadLink.style.display = 'none'
  const moveVideoBtn = el('button', { class: 'move-btn' }, ['Save to Video Folder'])
  moveVideoBtn.style.display = 'none'
  const moveFolderBtn = el('button', { class: 'move-btn' }, ['Save to Folder…'])
  moveFolderBtn.style.display = 'none'
  const sidecarNote = el('div', { class: 'sidecar-note' })
  sidecarNote.style.display = 'none'
  const moveNote = el('div', { class: 'move-note' })
  moveNote.style.display = 'none'
  const logPane = el('div', { class: 'log-pane' })

  const root = el('div', { class: 'job-detail' }, [
    el('div', { class: 'detail-header' }, [stateLabel, cancelBtn, downloadLink, moveVideoBtn, moveFolderBtn]),
    sidecarNote,
    moveNote,
    el('div', { class: 'detail-progress' }, [progressBar, progressText]),
    logPane,
  ])
  container.replaceChildren(root)

  cancelBtn.addEventListener('click', async () => {
    cancelBtn.disabled = true
    try {
      await cancelJob(jobId)
    } catch (err) {
      appendLogLine('system', `Cancel failed: ${(err as Error).message}`)
      cancelBtn.disabled = false
    }
  })

  function showMoveNote(text: string, isError: boolean) {
    moveNote.textContent = text
    moveNote.className = `move-note ${isError ? 'move-note-error' : 'move-note-ok'}`
    moveNote.style.display = ''
  }

  async function doMove(destDir: string | undefined, trigger: HTMLButtonElement) {
    trigger.disabled = true
    try {
      const res = await moveSrt(jobId, destDir)
      showMoveNote(`Saved to: ${res.path}`, false)
    } catch (err) {
      showMoveNote(`Save failed: ${(err as Error).message}`, true)
    } finally {
      trigger.disabled = false
    }
  }

  moveVideoBtn.addEventListener('click', () => doMove(undefined, moveVideoBtn))
  moveFolderBtn.addEventListener('click', () => {
    openFolderPicker((destDir) => doMove(destDir, moveFolderBtn))
  })

  function appendLogLine(stream: string, text: string) {
    logPane.append(el('div', { class: `log-line log-${stream}` }, [text]))
    logPane.scrollTop = logPane.scrollHeight
  }

  interface ApplyStateOpts {
    error?: string
    srtAvailable?: boolean
    sidecarPath?: string
    sidecarError?: string
  }

  function applyState(state: string, opts: ApplyStateOpts = {}) {
    stateLabel.textContent = STATE_LABEL[state] ?? state
    stateLabel.className = `detail-state detail-state-${state}`
    const terminal = TERMINAL_STATES.has(state)
    cancelBtn.style.display = terminal ? 'none' : ''
    if (opts.srtAvailable) {
      downloadLink.setAttribute('href', srtDownloadUrl(jobId))
      downloadLink.style.display = ''
      moveVideoBtn.style.display = ''
      moveFolderBtn.style.display = ''
    }
    if (opts.sidecarPath) {
      sidecarNote.textContent = `Saved next to video: ${opts.sidecarPath}`
      sidecarNote.className = 'sidecar-note sidecar-ok'
      sidecarNote.style.display = ''
    } else if (opts.sidecarError) {
      sidecarNote.textContent = `Could not save next to video (${opts.sidecarError}) -- use Download instead`
      sidecarNote.className = 'sidecar-note sidecar-error'
      sidecarNote.style.display = ''
    }
    if (opts.error) appendLogLine('system', `Error: ${opts.error}`)
    if (terminal) onTerminal?.()
  }

  getJob(jobId)
    .then((job: Job) =>
      applyState(job.state, {
        error: job.error,
        srtAvailable: Boolean(job.srtPath),
        sidecarPath: job.srtSidecarPath,
        sidecarError: job.srtSidecarError,
      }),
    )
    .catch((err) => appendLogLine('system', `Failed to load job: ${(err as Error).message}`))

  const closeStream = openJobStream(jobId, {
    onLog: (e: LogEvent) => appendLogLine(e.stream, e.text),
    onProgress: (e: ProgressEvent) => {
      if (e.percent !== null) {
        progressFill.style.width = `${e.percent}%`
        progressText.textContent = `${e.percent}%`
      } else {
        progressText.textContent = e.raw
      }
    },
    onState: (e: StateEvent) =>
      applyState(e.state, {
        error: e.error,
        srtAvailable: e.srtAvailable,
        sidecarPath: e.srtSidecarPath,
        sidecarError: e.srtSidecarError,
      }),
  })

  return {
    destroy() {
      closeStream()
    },
  }
}
