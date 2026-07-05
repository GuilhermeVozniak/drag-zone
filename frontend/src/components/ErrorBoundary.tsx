import { Component, type ErrorInfo, type ReactNode } from "react"

interface ErrorBoundaryState {
  error: Error | null
}

/**
 * Catches render/runtime errors from the UI tree and shows the message
 * instead of letting the whole (transparent) window blank out to grey.
 */
export class ErrorBoundary extends Component<{ children: ReactNode }, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("DragZone UI error:", error, info.componentStack)
  }

  render() {
    if (this.state.error) {
      return (
        <div className="flex h-screen flex-col gap-2 overflow-auto rounded-2xl border border-red-500/30 bg-neutral-900 p-4 text-neutral-200">
          <p className="text-sm font-semibold text-red-400">Something went wrong</p>
          <pre className="whitespace-pre-wrap text-[11px] text-neutral-400">
            {this.state.error.message}
          </pre>
        </div>
      )
    }
    return this.props.children
  }
}
