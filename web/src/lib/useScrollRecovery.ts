import { useEffect } from 'react'

const OVERLAY_SELECTOR = '[role="dialog"], [role="alertdialog"], [role="menu"], [role="listbox"], [data-radix-popper-content-wrapper]'

function overlayOpen(): boolean {
  return document.querySelector(OVERLAY_SELECTOR) !== null
}

function releaseStaleLocks() {
  if (overlayOpen()) return
  if (document.body.style.pointerEvents === 'none') document.body.style.pointerEvents = ''
  if (document.body.hasAttribute('data-scroll-locked')) document.body.removeAttribute('data-scroll-locked')
  if (window.scrollX !== 0 || window.scrollY !== 0) window.scrollTo(0, 0)
}

export function useScrollRecovery(routeKey: string) {
  useEffect(releaseStaleLocks, [routeKey])

  useEffect(() => {
    const viewport = window.visualViewport
    const recover = () => window.requestAnimationFrame(releaseStaleLocks)
    viewport?.addEventListener('resize', recover)
    window.addEventListener('resize', recover)
    window.addEventListener('pageshow', recover)
    window.addEventListener('orientationchange', recover)
    window.addEventListener('focusout', recover)
    document.addEventListener('visibilitychange', recover)
    return () => {
      viewport?.removeEventListener('resize', recover)
      window.removeEventListener('resize', recover)
      window.removeEventListener('pageshow', recover)
      window.removeEventListener('orientationchange', recover)
      window.removeEventListener('focusout', recover)
      document.removeEventListener('visibilitychange', recover)
    }
  }, [])
}
