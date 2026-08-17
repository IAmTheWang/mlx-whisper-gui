export type JobState = 'queued' | 'running' | 'done' | 'failed' | 'cancelled'

export interface Job {
  id: string
  videoPath: string
  model: string
  language: string
  outputDir: string
  state: JobState
  createdAt: string
  startedAt?: string
  finishedAt?: string
  error?: string
  srtPath?: string
  srtSidecarPath?: string
  srtSidecarError?: string
}

export interface BrowseEntry {
  name: string
  path: string
  isDir: boolean
  size: number
  modTime: string
  ext: string
  isVideo: boolean
}

export interface Bookmark {
  label: string
  path: string
}

export interface BrowseResponse {
  path: string
  parent?: string
  entries: BrowseEntry[]
  bookmarks: Bookmark[]
}

export interface ModelInfo {
  id: string
  label: string
  cached: boolean
}

export interface LanguageInfo {
  code: string
  label: string
}

export interface ModelsResponse {
  models: ModelInfo[]
  languages: LanguageInfo[]
}

export interface SettingsResponse {
  mlxWhisperPath: string
  mlxResolvedVia: 'config' | 'path' | 'fallback' | 'none'
  ffmpegPath: string
  ffmpegResolvedVia: 'config' | 'path' | 'fallback' | 'none'
}

export interface LogEvent {
  seq: number
  stream: string
  text: string
}

export interface ProgressEvent {
  percent: number | null
  raw: string
}

export interface StateEvent {
  state: JobState
  error?: string
  srtAvailable: boolean
  srtSidecarPath?: string
  srtSidecarError?: string
}
