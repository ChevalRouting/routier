import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { Toaster } from '@/components/ui/sonner'
import { ThemeContext, useThemeState } from '@/lib/useTheme'
import App from './App.tsx'
import './index.css'

function Root() {
  const themeValue = useThemeState()
  return (
    <ThemeContext.Provider value={themeValue}>
      <App />
      <Toaster theme={themeValue.theme} />
    </ThemeContext.Provider>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <Root />
    </BrowserRouter>
  </StrictMode>,
)
