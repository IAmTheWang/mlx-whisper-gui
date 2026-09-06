import type { EngineID, LanguageInfo, ModelInfo } from '../types'
import { el, clear } from '../dom'

export interface JobFormHandlers {
  onEngineChange: (engine: EngineID) => void
  onStart: (engine: EngineID, model: string, language: string) => void
}

export interface JobFormController {
  setSelectionCount(n: number): void
  setModels(models: ModelInfo[]): void
}

const DEFAULT_MLX_MODEL = 'mlx-community/whisper-large-v3-mlx'

export function renderJobForm(
  container: HTMLElement,
  initialEngine: EngineID,
  models: ModelInfo[],
  languages: LanguageInfo[],
  handlers: JobFormHandlers,
): JobFormController {
  const engineSelect = el('select', {}, [
    el('option', { value: 'mlx' }, ['mlx_whisper']),
    el('option', { value: 'whispercpp' }, ['whisper.cpp']),
  ])
  engineSelect.value = initialEngine

  // Model options depend on the selected engine and are swapped out wholesale
  // via setModels -- this slot holds whichever <select> is current.
  const modelSlot = el('span', {})
  let modelSelect: HTMLSelectElement

  const languageSelect = el(
    'select',
    {},
    languages.map((l) => el('option', { value: l.code }, [l.label])),
  )
  const jaIndex = languages.findIndex((l) => l.code === 'ja')
  if (jaIndex >= 0) languageSelect.selectedIndex = jaIndex

  const countLabel = el('span', { class: 'selection-count' }, ['No files selected'])
  const startBtn = el('button', { class: 'start-btn' }, ['Start Transcription'])
  startBtn.disabled = true

  let selectionCount = 0

  function updateStartDisabled() {
    startBtn.disabled = selectionCount === 0 || !modelSelect.value
  }

  function buildModelSelect(models: ModelInfo[]) {
    if (models.length === 0) {
      modelSelect = el('select', {}, [
        el('option', { value: '', disabled: true, selected: true }, ['No models found -- configure one in Settings']),
      ])
      modelSelect.disabled = true
    } else {
      modelSelect = el(
        'select',
        {},
        models.map((m) => el('option', { value: m.id }, [`${m.label}${m.cached ? '' : ' (needs download)'}`])),
      )
      const defaultIndex = models.findIndex((m) => m.id === DEFAULT_MLX_MODEL)
      if (defaultIndex >= 0) modelSelect.selectedIndex = defaultIndex
    }
    modelSelect.addEventListener('change', updateStartDisabled)
    clear(modelSlot)
    modelSlot.append(modelSelect)
  }

  buildModelSelect(models)

  engineSelect.addEventListener('change', () => {
    handlers.onEngineChange(engineSelect.value as EngineID)
  })

  startBtn.addEventListener('click', () => {
    handlers.onStart(engineSelect.value as EngineID, modelSelect.value, languageSelect.value)
  })

  container.append(
    el('div', { class: 'job-form' }, [
      el('label', {}, ['Engine:', engineSelect]),
      el('label', {}, ['Model:', modelSlot]),
      el('label', {}, ['Language:', languageSelect]),
      countLabel,
      startBtn,
    ]),
  )

  return {
    setSelectionCount(n: number) {
      selectionCount = n
      countLabel.textContent = n > 0 ? `${n} file${n === 1 ? '' : 's'} selected` : 'No files selected'
      updateStartDisabled()
    },
    setModels(models: ModelInfo[]) {
      buildModelSelect(models)
      updateStartDisabled()
    },
  }
}
