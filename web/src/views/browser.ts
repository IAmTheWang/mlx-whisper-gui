import type { Bookmark, BrowseEntry, BrowseResponse } from '../types'
import { browseDir } from '../api'
import { el, clear } from '../dom'
import { parseSortState, renderSortRow, sortEntries, type SortState } from '../sortControls'

const LAST_PATH_KEY = 'whisper-gui:lastPath'
const SORT_KEY = 'whisper-gui:browserSort'
const SORT_OPTIONS: { key: SortState['key']; label: string }[] = [
  { key: 'name', label: 'Name' },
  { key: 'date', label: 'Date' },
  { key: 'type', label: 'Type' },
]

export interface BrowserHandlers {
  onSelectionChange: (paths: string[]) => void
}

export interface BrowserController {
  clearSelection(): void
  getEntry(path: string): BrowseEntry | undefined
}

export function renderBrowser(container: HTMLElement, handlers: BrowserHandlers): BrowserController {
  const selected = new Set<string>()
  // Entries seen across every directory visited this session, keyed by path
  // -- selections can span multiple directories, but `lastData` only holds
  // the currently displayed one, so this is the only place still holding
  // metadata (like hasSrt) for a selected file after the user navigates away.
  const knownEntries = new Map<string, BrowseEntry>()
  let lastData: BrowseResponse | undefined
  let sortState: SortState = parseSortState(localStorage.getItem(SORT_KEY), ['name', 'date', 'type'])

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
    for (const entry of entries) knownEntries.set(entry.path, entry)
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

    root.append(
      renderSortRow(SORT_OPTIONS, sortState, (next) => {
        sortState = next
        localStorage.setItem(SORT_KEY, `${next.key}:${next.dir}`)
        renderLoaded()
      }),
    )

    const list = el('div', { class: 'entry-list' })
    for (const entry of sortEntries(entries, sortState)) {
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
    getEntry(path) {
      return knownEntries.get(path)
    },
  }
}
