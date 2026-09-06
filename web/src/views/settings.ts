import type { EngineID, SettingsResponse } from '../types'
import { getSettings, saveSettings } from '../api'
import { el, clear } from '../dom'
import { openFolderPicker } from './folderPicker'

// A modal overlay, same pattern as folderPicker.ts's openFolderPicker --
// Settings is opened on demand rather than living in the main 4-pane layout,
// since it's used rarely (initial setup, or when auto-detection fails).
export function openSettingsModal(onSaved?: () => void): void {
  const panel = el('div', { class: 'settings-panel' })
  const overlay = el(
    'div',
    {
      class: 'settings-overlay',
      onclick: (e: MouseEvent) => {
        if (e.target === overlay) close()
      },
    },
    [panel],
  )
  document.body.append(overlay)

  function close() {
    overlay.remove()
  }

  function resolvedBadge(via: string): HTMLElement {
    return el('span', { class: `resolved-badge resolved-${via}` }, [via])
  }

  async function load() {
    clear(panel)
    panel.append(el('div', { class: 'settings-status' }, ['Loading…']))
    let settings: SettingsResponse
    try {
      settings = await getSettings()
    } catch (err) {
      clear(panel)
      panel.append(el('div', { class: 'settings-error' }, [(err as Error).message]))
      return
    }
    renderLoaded(settings)
  }

  function renderLoaded(settings: SettingsResponse, savedMessage?: string) {
    clear(panel)

    const mlxInput = el('input', { value: settings.mlxWhisperPath, placeholder: '/path/to/mlx_whisper' })
    const ffmpegInput = el('input', { value: settings.ffmpegPath, placeholder: '/path/to/ffmpeg' })
    const whisperCliInput = el('input', { value: settings.whisperCliPath, placeholder: '/path/to/whisper-cli' })
    const modelDirInput = el('input', {
      value: settings.whisperCppModelDir,
      placeholder: '/path/to/whisper.cpp/models',
    })
    const browseModelDirBtn = el('button', { class: 'settings-browse-btn' }, ['Browse…'])
    browseModelDirBtn.addEventListener('click', () => {
      openFolderPicker((path) => {
        modelDirInput.value = path
      })
    })

    const engineSelect = el('select', {}, [
      el('option', { value: '' }, ['Auto (use whichever is available)']),
      el('option', { value: 'mlx' }, ['mlx_whisper']),
      el('option', { value: 'whispercpp' }, ['whisper.cpp']),
    ])
    engineSelect.value = settings.defaultEngine

    const saveBtn = el('button', { class: 'settings-save-btn' }, ['Save'])
    const resultNote = el('div', { class: 'settings-result' })
    resultNote.style.display = 'none'
    if (savedMessage) {
      resultNote.textContent = savedMessage
      resultNote.className = 'settings-result settings-result-ok'
      resultNote.style.display = ''
    }

    saveBtn.addEventListener('click', async () => {
      saveBtn.disabled = true
      resultNote.style.display = 'none'
      try {
        const updated = await saveSettings({
          mlxWhisperPath: mlxInput.value,
          ffmpegPath: ffmpegInput.value,
          whisperCliPath: whisperCliInput.value,
          whisperCppModelDir: modelDirInput.value,
          defaultEngine: engineSelect.value as EngineID | '',
        })
        // Rebuilds the whole panel (fresh inputs/badges/buttons reflecting
        // the saved state), so the "Saved." message must be handed to the
        // new render pass rather than written onto this now-detached note.
        renderLoaded(updated, 'Saved.')
        onSaved?.()
      } catch (err) {
        resultNote.textContent = (err as Error).message
        resultNote.className = 'settings-result settings-result-error'
        resultNote.style.display = ''
        saveBtn.disabled = false
      }
    })

    panel.append(
      el('div', { class: 'settings-header' }, [
        el('span', {}, ['Settings']),
        el('button', { class: 'settings-close-btn', onclick: close }, ['×']),
      ]),
      el('label', { class: 'settings-field' }, ['mlx_whisper path', mlxInput, resolvedBadge(settings.mlxResolvedVia)]),
      el('label', { class: 'settings-field' }, ['ffmpeg path', ffmpegInput, resolvedBadge(settings.ffmpegResolvedVia)]),
      el('label', { class: 'settings-field' }, [
        'whisper-cli path',
        whisperCliInput,
        resolvedBadge(settings.whisperCliResolvedVia),
      ]),
      el('label', { class: 'settings-field' }, ['whisper.cpp model directory', modelDirInput, browseModelDirBtn]),
      el('label', { class: 'settings-field' }, ['Default engine', engineSelect]),
      saveBtn,
      resultNote,
    )
  }

  void load()
}
