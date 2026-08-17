import type { BrowseResponse, Job, ModelsResponse, SettingsResponse } from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
  })
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`
    try {
      const body = await res.json()
      if (body && typeof body.error === 'string') message = body.error
    } catch {
      // fall back to the plain status text above
    }
    throw new Error(message)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export function browseDir(path?: string): Promise<BrowseResponse> {
  const qs = path ? `?path=${encodeURIComponent(path)}` : ''
  return request(`/api/browse${qs}`)
}

export function listModels(): Promise<ModelsResponse> {
  return request('/api/models')
}

export function getSettings(): Promise<SettingsResponse> {
  return request('/api/settings')
}

export function saveSettings(update: { mlxWhisperPath?: string; ffmpegPath?: string }): Promise<SettingsResponse> {
  return request('/api/settings', { method: 'PUT', body: JSON.stringify(update) })
}

export function createJob(req: { videoPath: string; model: string; language: string }): Promise<Job> {
  return request('/api/jobs', { method: 'POST', body: JSON.stringify(req) })
}

export function listJobs(): Promise<Job[]> {
  return request('/api/jobs')
}

export function getJob(id: string): Promise<Job> {
  return request(`/api/jobs/${encodeURIComponent(id)}`)
}

export function cancelJob(id: string): Promise<Job> {
  return request(`/api/jobs/${encodeURIComponent(id)}/cancel`, { method: 'POST' })
}

export function srtDownloadUrl(id: string): string {
  return `/api/jobs/${encodeURIComponent(id)}/srt`
}
