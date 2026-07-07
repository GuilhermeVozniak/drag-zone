import { describe, expect, it } from 'vitest'
import { uiScale, type Settings } from '@/lib/backend'

const settings = (gridSize: number): Settings => ({ gridSize }) as Settings

describe('uiScale', () => {
  it('defaults to the 33% grid size when settings are null', () => {
    expect(uiScale(null)).toBeCloseTo(0.998, 3) // 0.8 + 0.33*0.6
  })
  it('maps gridSize 0 to the minimum scale 0.8', () => {
    expect(uiScale(settings(0))).toBeCloseTo(0.8, 5)
  })
  it('maps gridSize 100 to the maximum scale 1.4', () => {
    expect(uiScale(settings(100))).toBeCloseTo(1.4, 5)
  })
  it('clamps gridSize above 100', () => {
    expect(uiScale(settings(150))).toBeCloseTo(1.4, 5)
  })
  it('clamps negative gridSize up to 0.8', () => {
    expect(uiScale(settings(-20))).toBeCloseTo(0.8, 5)
  })
})
