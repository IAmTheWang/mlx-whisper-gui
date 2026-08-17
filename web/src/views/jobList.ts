import type { Job } from '../types'
import { listJobs } from '../api'
import { el, clear } from '../dom'

export interface JobListHandlers {
  onSelect: (id: string) => void
}

export interface JobListController {
  refresh(): Promise<void>
}

const STATE_LABEL: Record<string, string> = {
  queued: 'Queued',
  running: 'Running',
  done: 'Done',
  failed: 'Failed',
  cancelled: 'Cancelled',
}

export function renderJobList(container: HTMLElement, handlers: JobListHandlers): JobListController {
  const root = el('div', { class: 'job-list' })
  container.append(root)

  async function refresh() {
    let jobs: Job[]
    try {
      jobs = await listJobs()
    } catch (err) {
      root.replaceChildren(el('div', { class: 'job-list-error' }, [(err as Error).message]))
      return
    }

    clear(root)
    if (jobs.length === 0) {
      root.append(el('div', { class: 'job-list-empty' }, ['No jobs yet']))
      return
    }
    for (const job of jobs) {
      const filename = job.videoPath.split('/').pop() ?? job.videoPath
      root.append(
        el('div', { class: `job-row job-row-${job.state}`, onclick: () => handlers.onSelect(job.id) }, [
          el('span', { class: 'job-row-name' }, [filename]),
          el('span', { class: 'job-row-state' }, [STATE_LABEL[job.state] ?? job.state]),
        ]),
      )
    }
  }

  refresh()
  return { refresh }
}
