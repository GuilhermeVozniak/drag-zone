import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend, type Target } from '@/lib/backend'
import { useTargetShortcuts } from '@/hooks/useTargetShortcuts'

vi.mock('@/lib/backend')

const target = (id: string, shortcut?: string): Target => ({ id, shortcut }) as Target

function press(key: string, opts: Partial<KeyboardEventInit> = {}) {
  window.dispatchEvent(
    new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...opts }),
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('useTargetShortcuts', () => {
  it('clicks the target whose shortcut matches the key (case-insensitive)', () => {
    renderHook(() => useTargetShortcuts([target('t1', 'F'), target('t2', 'G')]))
    press('f')
    expect(backend.click).toHaveBeenCalledWith('t1')
  })
  it('ignores keystrokes held with a modifier', () => {
    renderHook(() => useTargetShortcuts([target('t1', 'F')]))
    press('f', { metaKey: true })
    press('f', { ctrlKey: true })
    press('f', { altKey: true })
    expect(backend.click).not.toHaveBeenCalled()
  })
  it('ignores keystrokes aimed at an input element', () => {
    const input = document.createElement('input')
    document.body.appendChild(input)
    renderHook(() => useTargetShortcuts([target('t1', 'F')]))
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'f', bubbles: true }))
    expect(backend.click).not.toHaveBeenCalled()
  })
  it('does nothing when no shortcut matches', () => {
    renderHook(() => useTargetShortcuts([target('t1', 'F')]))
    press('z')
    expect(backend.click).not.toHaveBeenCalled()
  })
  it('ignores non-single-character keys such as Enter', () => {
    renderHook(() => useTargetShortcuts([target('t1', 'F')]))
    press('Enter')
    expect(backend.click).not.toHaveBeenCalled()
  })
  it('detaches its keydown listener on unmount', () => {
    const { unmount } = renderHook(() => useTargetShortcuts([target('t1', 'F')]))
    unmount()
    press('f')
    expect(backend.click).not.toHaveBeenCalled()
  })
})
