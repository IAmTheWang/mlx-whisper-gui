import type { LanguageInfo, ModelInfo } from '../types'
import { el } from '../dom'

export interface JobFormHandlers {
  onStart: (model: string, language: string) => void
}

export interface JobFormController {
  setSelectionCount(n: number): void
}

export function renderJobForm(
  container: HTMLElement,
  models: ModelInfo[],
  languages: LanguageInfo[],
  handlers: JobFormHandlers,
): JobFormController {
  const modelSelect = el(
    'select',
    {},
    models.map((m) => el('option', { value: m.id }, [`${m.label}${m.cached ? '' : ' (needs download)'}`])),
  )

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

  startBtn.addEventListener('click', () => {
    handlers.onStart(modelSelect.value, languageSelect.value)
  })

  container.append(
    el('div', { class: 'job-form' }, [
      el('label', {}, ['Model:', modelSelect]),
      el('label', {}, ['Language:', languageSelect]),
      countLabel,
      startBtn,
    ]),
  )

  return {
    setSelectionCount(n: number) {
      countLabel.textContent = n > 0 ? `${n} file${n === 1 ? '' : 's'} selected` : 'No files selected'
      startBtn.disabled = n === 0
    },
  }
}
