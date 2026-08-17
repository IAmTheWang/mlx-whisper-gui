type Props = Record<string, unknown>

export function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  props: Props = {},
  children: (Node | string)[] = [],
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag)
  for (const [key, value] of Object.entries(props)) {
    if (value === undefined || value === null) continue
    if (key === 'class') {
      node.className = String(value)
    } else if (key.startsWith('on') && typeof value === 'function') {
      node.addEventListener(key.slice(2).toLowerCase(), value as EventListener)
    } else {
      node.setAttribute(key, String(value))
    }
  }
  node.append(...children)
  return node
}

export function clear(node: HTMLElement): void {
  node.replaceChildren()
}
