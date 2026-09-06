import './style.css'
import { createJob, getSettings, listModels } from './api'
import type { EngineID } from './types'
import { el } from './dom'
import { renderBrowser } from './views/browser'
import { renderJobForm, type JobFormController } from './views/jobForm'
import { renderJobList } from './views/jobList'
import { renderJobDetail, type JobDetailController } from './views/jobDetail'
import { openSettingsModal } from './views/settings'

async function main() {
  const app = document.querySelector<HTMLDivElement>('#app')
  if (!app) throw new Error('#app root not found')
  app.replaceChildren()

  const browserPane = el('div', { class: 'pane-browser' })
  const formPane = el('div', { class: 'pane-form' })
  const listPane = el('div', { class: 'pane-list' })
  const detailPane = el('div', { class: 'pane-detail' })
  const settingsBtn = el('button', { class: 'settings-open-btn' }, ['⚙ Settings'])

  app.append(
    el('div', { class: 'app-header' }, [settingsBtn]),
    el('div', { class: 'app-layout' }, [
      el('div', { class: 'col-left' }, [browserPane, formPane]),
      el('div', { class: 'col-right' }, [listPane, detailPane]),
    ]),
  )

  let selectedPaths: string[] = []
  let form: JobFormController | undefined
  let detailController: JobDetailController | undefined
  let currentEngine: EngineID = 'mlx'

  settingsBtn.addEventListener('click', () => {
    openSettingsModal(() => {
      void refreshModels(currentEngine)
    })
  })

  const jobList = renderJobList(listPane, {
    onSelect: (id) => openJobDetail(id),
  })

  function openJobDetail(id: string) {
    detailController?.destroy()
    detailController = renderJobDetail(detailPane, id, () => {
      void jobList.refresh()
    })
  }

  const browser = renderBrowser(browserPane, {
    onSelectionChange: (paths) => {
      selectedPaths = paths
      form?.setSelectionCount(paths.length)
    },
  })

  async function refreshModels(engine: EngineID) {
    const { models } = await listModels(engine)
    form?.setModels(models)
  }

  const settings = await getSettings()
  if (settings.defaultEngine === 'mlx' || settings.defaultEngine === 'whispercpp') {
    currentEngine = settings.defaultEngine
  } else {
    // defaultEngine unset -- auto-pick whichever binary is actually resolvable,
    // so a fresh Mac with only `brew install whisper-cpp` (no Python/mlx_whisper
    // set up) doesn't land on an engine with zero usable models by default.
    const mlxAvailable = settings.mlxResolvedVia !== 'none'
    const whisperCliAvailable = settings.whisperCliResolvedVia !== 'none'
    if (!mlxAvailable && whisperCliAvailable) currentEngine = 'whispercpp'
  }

  const { models, languages } = await listModels(currentEngine)
  form = renderJobForm(formPane, currentEngine, models, languages, {
    onEngineChange: (engine) => {
      currentEngine = engine
      void refreshModels(engine)
    },
    onStart: async (engine, model, language) => {
      const paths = [...selectedPaths]

      const overwriteNames = paths
        .map((p) => browser.getEntry(p))
        .filter((e): e is NonNullable<typeof e> => !!e?.hasSrt)
        .map((e) => e.name)
      if (overwriteNames.length > 0) {
        const proceed = confirm(
          `These files already have a subtitle, which will be overwritten:\n\n${overwriteNames.join('\n')}\n\nContinue?`,
        )
        if (!proceed) return
      }

      let lastJobId: string | undefined
      for (const videoPath of paths) {
        try {
          const job = await createJob({ videoPath, engine, model, language })
          lastJobId = job.id
        } catch (err) {
          alert(`Failed to create job (${videoPath}): ${(err as Error).message}`)
        }
      }
      browser.clearSelection()
      form?.setSelectionCount(0)
      await jobList.refresh()
      if (lastJobId) openJobDetail(lastJobId)
    },
  })
}

main().catch((err) => {
  console.error(err)
  document.body.append(el('pre', { class: 'fatal-error' }, [String(err)]))
})
