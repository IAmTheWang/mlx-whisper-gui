import type { BrowseEntry } from './types'
import { el } from './dom'

export type SortKey = 'name' | 'date' | 'type'
export type SortDir = 'asc' | 'desc'

export interface SortState {
  key: SortKey
  dir: SortDir
}

const DEFAULT_SORT: SortState = { key: 'name', dir: 'asc' }

function compareByKey(a: BrowseEntry, b: BrowseEntry, key: SortKey): number {
  switch (key) {
    case 'name':
      return a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
    case 'date': {
      const timeA = Date.parse(a.modTime) || 0
      const timeB = Date.parse(b.modTime) || 0
      return timeA - timeB
    }
    case 'type': {
      const extA = a.ext || ''
      const extB = b.ext || ''
      const diff = extA.localeCompare(extB, undefined, { sensitivity: 'base' })
      return diff !== 0 ? diff : a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
    }
  }
}

// Folders are always pinned above files, regardless of sortDir -- only the
// ordering *within* each group follows the chosen key/direction. Matches
// Finder's default "keep folders on top" behavior.
export function sortEntries(entries: BrowseEntry[], state: SortState): BrowseEntry[] {
  const mult = state.dir === 'asc' ? 1 : -1
  return [...entries].sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
    return mult * compareByKey(a, b, state.key)
  })
}

export function parseSortState(raw: string | null, validKeys: SortKey[], fallback: SortState = DEFAULT_SORT): SortState {
  if (!raw) return fallback
  const [key, dir] = raw.split(':')
  if (validKeys.includes(key as SortKey) && (dir === 'asc' || dir === 'desc')) {
    return { key: key as SortKey, dir }
  }
  return fallback
}

export function renderSortRow(
  options: { key: SortKey; label: string }[],
  initial: SortState,
  onChange: (next: SortState) => void,
): HTMLElement {
  let state = initial
  const buttons = new Map<SortKey, HTMLElement>()

  function labelFor(opt: { key: SortKey; label: string }): string {
    if (state.key !== opt.key) return opt.label
    return `${opt.label} ${state.dir === 'asc' ? '▲' : '▼'}`
  }

  function refresh() {
    for (const opt of options) {
      const btn = buttons.get(opt.key)
      if (!btn) continue
      btn.textContent = labelFor(opt)
      btn.classList.toggle('sort-btn-active', state.key === opt.key)
      btn.setAttribute('aria-pressed', String(state.key === opt.key))
    }
  }

  const row = el(
    'div',
    { class: 'sort-row' },
    options.map((opt) => {
      const btn = el(
        'button',
        {
          type: 'button',
          class: 'sort-btn',
          onclick: () => {
            state = state.key === opt.key ? { key: opt.key, dir: state.dir === 'asc' ? 'desc' : 'asc' } : { key: opt.key, dir: 'asc' }
            refresh()
            onChange(state)
          },
        },
        [labelFor(opt)],
      )
      buttons.set(opt.key, btn)
      return btn
    }),
  )

  refresh()
  return row
}
