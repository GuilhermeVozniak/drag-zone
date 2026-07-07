import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend } from '@/lib/backend'
import { __fireEvent, __resetBackendMock } from '@/lib/__mocks__/backend'
import { InputRequestDialog } from '@/features/tasks/InputRequestDialog'

vi.mock('@/lib/backend')

beforeEach(() => {
  vi.clearAllMocks()
  __resetBackendMock()
})

function fireRequest(req: Record<string, unknown>) {
  act(() => __fireEvent('input:request', req))
}

describe('InputRequestDialog', () => {
  it('answers a text prompt with the typed value on OK', async () => {
    const user = userEvent.setup()
    render(<InputRequestDialog />)
    fireRequest({ id: 'r1', title: 'Name', prompt: 'New name?' })
    await user.type(screen.getByRole('textbox'), 'foo')
    await user.click(screen.getByRole('button', { name: 'OK' }))
    expect(backend.answerInput).toHaveBeenCalledWith('r1', 'foo', true)
  })

  it('answers not-answered on Cancel', async () => {
    const user = userEvent.setup()
    render(<InputRequestDialog />)
    fireRequest({ id: 'r1', title: 'Name', prompt: 'New name?' })
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(backend.answerInput).toHaveBeenCalledWith('r1', '', false)
  })

  it('answers a choice prompt with the picked label', async () => {
    const user = userEvent.setup()
    render(<InputRequestDialog />)
    fireRequest({
      id: 'r2',
      title: 'File exists',
      prompt: 'a.txt already exists',
      choices: ['Keep Both', 'Replace', 'Stop'],
    })
    await user.click(screen.getByRole('button', { name: 'Replace' }))
    expect(backend.answerInput).toHaveBeenCalledWith('r2', 'Replace', true)
  })
})
