import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Onboarding } from '@/features/onboarding/Onboarding'

describe('Onboarding', () => {
  it('starts on the welcome slide with Back hidden', () => {
    render(<Onboarding onDone={vi.fn()} />)
    expect(screen.getByText('Welcome to DragZone')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Back/ })).toHaveClass('invisible')
  })

  it('advances through slides with Next', async () => {
    const user = userEvent.setup()
    render(<Onboarding onDone={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: /Next/ }))
    expect(screen.getByText('Drop files onto actions')).toBeInTheDocument()
  })

  it('jumps to a slide via its dot', async () => {
    const user = userEvent.setup()
    render(<Onboarding onDone={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: 'Slide 3' }))
    expect(screen.getByText('Stash in the Drop Bar')).toBeInTheDocument()
  })

  it('calls onDone from Skip', async () => {
    const user = userEvent.setup()
    const onDone = vi.fn()
    render(<Onboarding onDone={onDone} />)
    await user.click(screen.getByRole('button', { name: 'Skip' }))
    expect(onDone).toHaveBeenCalled()
  })

  it('calls onDone from Get Started on the last slide', async () => {
    const user = userEvent.setup()
    const onDone = vi.fn()
    render(<Onboarding onDone={onDone} />)
    await user.click(screen.getByRole('button', { name: 'Slide 5' }))
    await user.click(screen.getByRole('button', { name: 'Get Started' }))
    expect(onDone).toHaveBeenCalled()
  })
})
