import { Component, type ErrorInfo, type ReactNode } from 'react'
import { Button } from 'cheval-ui'

interface PageErrorBoundaryProps {
  children: ReactNode
}

interface PageErrorBoundaryState {
  error: Error | null
}

export default class PageErrorBoundary extends Component<PageErrorBoundaryProps, PageErrorBoundaryState> {
  state: PageErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): PageErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Failed to display page', error, info.componentStack)
  }

  reload = () => window.location.reload()

  render() {
    if (!this.state.error) return this.props.children

    return (
      <div role="alert" className="space-y-3 rounded-lg border border-border bg-card p-4">
        <h2 className="text-sm font-semibold">Unable to Load This Page</h2>
        <p className="text-sm text-muted-foreground">The app may have been updated, or the connection interrupted. Reload to try again. Unsaved edits will be lost.</p>
        <Button size="sm" onClick={this.reload}>Reload Page</Button>
      </div>
    )
  }
}
