import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { ThemeProvider, Toaster, useTheme } from 'cheval-ui'
import App from './App.tsx'
import 'cheval-ui/styles.css'
import './index.css'

function RoutierToaster() {
  const { theme } = useTheme()
  return <Toaster theme={theme} />
}

function Root() {
  return (
    <ThemeProvider storageKey="routier-theme" defaultTheme="dark">
      <App />
      <RoutierToaster />
    </ThemeProvider>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <Root />
    </BrowserRouter>
  </StrictMode>,
)
