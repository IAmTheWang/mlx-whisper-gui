import type { Bookmark, BrowseEntry, BrowseResponse } from '../types'
import { browseDir } from '../api'
import { el, clear } from '../dom'

const LAST_PATH_KEY = 'whisper-gui:lastPath'

export interface BrowserHandlers {
  onSelectionChange: (paths: string[]) => void
}

export interface BrowserController {
  clearSelection(): void
}

export function renderBrowser(container: HTMLElement, handlers: BrowserHandlers): BrowserController {
  const selected = new Set<string>()
  let lastData: BrowseResponse | undefined

  const root = el('div', { class: 'browser' })
  container.append(root)

  async function load(path?: string) {
    root.replaceChildren(el('div', { class: 'browser-status' }, ['Loading…']))
    try {
      lastData = await browseDir(path)
    } catch (err) {
      root.replaceChildren(el('div', { class: 'browser-error' }, [(err as Error).message]))
      return
    }
    localStorage.setItem(LAST_PATH_KEY, lastData.path)
    renderLoaded()
  }

  function renderLoaded() {
    if (!lastData) return
    const { path, parent, entries, bookmarks } = lastData
    clear(root)

    root.append(
      el(
        'div',
        { class: 'bookmarks' },
        bookmarks.map((b: Bookmark) => el('button', { class: 'bookmark-btn', onclick: () => load(b.path) }, [b.label])),
      ),
    )

    root.append(
      el('div', { class: 'breadcrumb-row' }, [
        parent
          ? el('button', { class: 'up-btn', onclick: () => load(parent) }, ['↑ Parent Directory'])
          : el('span', {}, []),
        el('span', { class: 'breadcrumb' }, [path]),
      ]),
    )

    const list = el('div', { class: 'entry-list' })
    for (const entry of entries) {
      list.append(renderEntry(entry))
    }
    root.append(list)
  }

  function renderEntry(entry: BrowseEntry): HTMLElement {
    if (entry.isDir) {
      return el('div', { class: 'entry entry-dir', onclick: () => load(entry.path) }, [
        el('span', { class: 'entry-icon' }, ['📁']),
        el('span', { class: 'entry-name' }, [entry.name]),
      ])
    }

    const checkbox = el('input', { type: 'checkbox' })
    checkbox.disabled = !entry.isVideo
    checkbox.checked = selected.has(entry.path)
    checkbox.addEventListener('change', () => {
      if (checkbox.checked) selected.add(entry.path)
      else selected.delete(entry.path)
      handlers.onSelectionChange([...selected])
    })

    // A real <label> wrapping the checkbox makes the whole row a click
    // target (standard HTML association, no JS needed) instead of only the
    // ~13px checkbox square itself.
    return el('label', { class: `entry entry-file${entry.isVideo ? ' entry-video' : ''}` }, [
      checkbox,
      el('span', { class: 'entry-icon' }, [entry.isVideo ? '🎬' : '📄']),
      el('span', { class: 'entry-name' }, [entry.name]),
    ])
  }

  load(localStorage.getItem(LAST_PATH_KEY) ?? undefined)

  return {
    clearSelection() {
      selected.clear()
      handlers.onSelectionChange([])
      renderLoaded()
    },
  }
}
