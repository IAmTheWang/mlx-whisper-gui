import type { Bookmark, BrowseEntry, BrowseResponse } from '../types'
import { browseDir } from '../api'
import { el, clear } from '../dom'

const LAST_DEST_KEY = 'whisper-gui:lastSrtDestDir'

// A directory-only variant of browser.ts's file browser, shown as a modal
// overlay so it can be opened on demand from job detail without taking over
// the main layout. Only directories are listed -- there is nothing to pick
// a file for here, the destination folder itself is the selection. Like
// browser.ts's own last-visited path, the last folder browsed to here is
// remembered in localStorage so the picker reopens where the user left off
// instead of always starting at Home.
export function openFolderPicker(onSelect: (path: string) => void): void {
  const panel = el('div', { class: 'folder-picker-panel' })
  const overlay = el('div', { class: 'folder-picker-overlay', onclick: (e: MouseEvent) => {
    if (e.target === overlay) close()
  } }, [panel])
  document.body.append(overlay)

  function close() {
    overlay.remove()
  }

  async function load(path?: string) {
    clear(panel)
    panel.append(el('div', { class: 'folder-picker-status' }, ['Loading…']))
    let data: BrowseResponse
    try {
      data = await browseDir(path)
    } catch (err) {
      clear(panel)
      panel.append(el('div', { class: 'folder-picker-error' }, [(err as Error).message]))
      return
    }
    localStorage.setItem(LAST_DEST_KEY, data.path)
    renderLoaded(data)
  }

  function renderLoaded(data: BrowseResponse) {
    const { path, parent, entries, bookmarks } = data
    clear(panel)

    panel.append(
      el(
        'div',
        { class: 'bookmarks' },
        bookmarks.map((b: Bookmark) => el('button', { class: 'bookmark-btn', onclick: () => load(b.path) }, [b.label])),
      ),
    )

    panel.append(
      el('div', { class: 'breadcrumb-row' }, [
        parent
          ? el('button', { class: 'up-btn', onclick: () => load(parent) }, ['↑ Parent Directory'])
          : el('span', {}, []),
        el('span', { class: 'breadcrumb' }, [path]),
      ]),
    )

    const list = el('div', { class: 'entry-list' })
    for (const entry of entries.filter((e: BrowseEntry) => e.isDir)) {
      list.append(
        el('div', { class: 'entry entry-dir', onclick: () => load(entry.path) }, [
          el('span', { class: 'entry-icon' }, ['📁']),
          el('span', { class: 'entry-name' }, [entry.name]),
        ]),
      )
    }
    panel.append(list)

    panel.append(
      el('div', { class: 'folder-picker-actions' }, [
        el('button', { class: 'folder-picker-cancel', onclick: close }, ['Cancel']),
        el(
          'button',
          {
            class: 'folder-picker-select',
            onclick: () => {
              onSelect(path)
              close()
            },
          },
          ['Select This Folder'],
        ),
      ]),
    )
  }

  load(localStorage.getItem(LAST_DEST_KEY) ?? undefined)
}
