import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

describe('test infrastructure', () => {
  it('renders JSX into jsdom and matches with jest-dom', () => {
    render(<div>hello dragzone</div>)
    expect(screen.getByText('hello dragzone')).toBeInTheDocument()
  })
})
