import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend } from '@/lib/backend'
import { useFileIcon } from '@/hooks/useFileIcon'

vi.mock('@/lib/backend')

const mockIcon = vi.mocked(backend.fileIcon)

beforeEach(() => {
  vi.clearAllMocks()
})

describe('useFileIcon', () => {
  it('returns null and never calls the backend for an undefined path', () => {
    const { result } = renderHook(() => useFileIcon(undefined))
    expect(result.current).toBeNull()
    expect(mockIcon).not.toHaveBeenCalled()
  })
  it('fetches and returns the base64 icon for a path', async () => {
    mockIcon.mockResolvedValue('BASE64DATA' as never)
    const { result } = renderHook(() => useFileIcon('/unique/one.txt'))
    await waitFor(() => expect(result.current).toBe('BASE64DATA'))
    expect(mockIcon).toHaveBeenCalledWith('/unique/one.txt')
  })
  it('serves a second hook from cache without a second backend call', async () => {
    mockIcon.mockResolvedValue('CACHED' as never)
    const first = renderHook(() => useFileIcon('/unique/two.txt'))
    await waitFor(() => expect(first.result.current).toBe('CACHED'))
    mockIcon.mockClear()
    const second = renderHook(() => useFileIcon('/unique/two.txt'))
    expect(second.result.current).toBe('CACHED')
    expect(mockIcon).not.toHaveBeenCalled()
  })
  it('maps an empty backend result to null', async () => {
    mockIcon.mockResolvedValue('' as never)
    const { result } = renderHook(() => useFileIcon('/unique/three.txt'))
    await waitFor(() => expect(mockIcon).toHaveBeenCalledWith('/unique/three.txt'))
    expect(result.current).toBeNull()
  })
})
