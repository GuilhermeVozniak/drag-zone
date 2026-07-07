import { vi } from 'vitest'

// Every binding is an async no-op by default; tests override with
// mockResolvedValue where they assert on a return.
const afn = () => vi.fn(async () => undefined)

export const backend = {
  settings: { get: afn(), set: afn() },
  actions: { specs: afn(), installBundle: afn(), openFolder: afn(), develop: afn() },
  grid: {
    list: afn(), add: afn(), addFromPaths: afn(), update: afn(),
    duplicate: afn(), remove: afn(), move: afn(),
  },
  drop: afn(),
  click: afn(),
  tasks: { list: afn(), dismiss: afn(), cancel: afn() },
  shares: { list: afn(), clear: afn(), open: afn() },
  playDropSound: afn(),
  dropBar: {
    list: afn(), add: afn(), remove: afn(), clear: afn(), consume: afn(),
    setLocked: afn(), rename: afn(), setPopOut: afn(), separate: afn(),
    combineAll: afn(), copyToClipboard: afn(), reveal: afn(), paste: afn(),
  },
  quickLook: afn(),
  answerInput: afn(),
  addons: { list: afn(), install: afn() },
  cli: { installed: afn(), install: afn() },
  updates: { check: afn(), version: afn() },
  dialogs: { chooseFolder: afn(), chooseApplication: afn() },
  dragOut: afn(),
  fileIcon: afn(),
  openURL: afn(),
  window: { hide: afn(), quit: afn(), about: afn() },
}

// --- event registry: each subscriber records its latest callback + unsub ---
export const __eventCbs: Record<string, ((...a: unknown[]) => void) | null> = {}
export const __unsub: Record<string, ReturnType<typeof vi.fn>> = {}

function sub(name: string) {
  return vi.fn((fn: (...a: unknown[]) => void) => {
    __eventCbs[name] = fn
    const u = vi.fn()
    __unsub[name] = u
    return u
  })
}

export const events = {
  onGridChanged: sub('grid:changed'),
  onTasksChanged: sub('tasks:changed'),
  onDropBarChanged: sub('dropbar:changed'),
  onOpenSettings: sub('settings:open'),
  onSpecsChanged: sub('specs:changed'),
  onDropBarPopOut: sub('dropbar:popout'),
  onInputRequest: sub('input:request'),
  onWindowVisibility: sub('window:visibility'),
  onWindowBeak: sub('window:beak'),
  onSharesChanged: sub('shares:changed'),
}

// --- test helpers ---
export function __fireEvent(name: string, ...args: unknown[]) {
  __eventCbs[name]?.(...args)
}
export function __resetBackendMock() {
  for (const k of Object.keys(__eventCbs)) __eventCbs[k] = null
}

// Real formula (mirrors config.Settings.Scale) so mocked consumers still work.
export function uiScale(s: { gridSize?: number } | null): number {
  const pct = Math.min(100, Math.max(0, s?.gridSize ?? 33))
  return 0.8 + (pct / 100) * 0.6
}
