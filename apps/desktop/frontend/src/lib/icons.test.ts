import { Archive, File, Wifi } from 'lucide-react'
import { describe, expect, it } from 'vitest'
import { iconFor, tileStyleFor } from '@/lib/icons'

describe('iconFor', () => {
  it('returns the mapped icon for a known name', () => {
    expect(iconFor('archive')).toBe(Archive)
    expect(iconFor('wifi')).toBe(Wifi)
  })
  it('falls back to File for an unknown name', () => {
    expect(iconFor('does-not-exist')).toBe(File)
  })
})

describe('tileStyleFor', () => {
  it('returns the branded style for a known action id', () => {
    const s = tileStyleFor('airdrop', 'wifi')
    expect(s.glyph).toBe(Wifi)
    expect(s.shape).toContain('rounded-full')
  })
  it('uses the icon name for the glyph when the action id is unknown', () => {
    const s = tileStyleFor('custom-thing', 'archive')
    expect(s.glyph).toBe(Archive)
    expect(s.shape).toContain('rounded-[14px]')
  })
  it('falls back to the File glyph when both id and icon name are unknown', () => {
    expect(tileStyleFor('custom-thing', 'mystery').glyph).toBe(File)
  })
})
