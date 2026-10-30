import { InstanceMenu } from '@/components/InstanceSwitcher'
import { AboutDialog } from '@/components/shell/AboutDialog'
import { APPEARANCE_OPTIONS } from '@/components/shell/MainMenu'
import { advancedMenus } from '@/lib/navigation'
import { clearToken } from '@/lib/utils'
import { PageHeader, PreferencesGroup, Row, Segmented, useTheme } from 'cheval-ui'
import { Info, LogOut, SlidersHorizontal } from 'lucide-react'
import { useState } from 'react'
import { Navigate, useNavigate, useParams } from 'react-router-dom'

const THEME_OPTIONS = APPEARANCE_OPTIONS.map(({ value, label }) => ({ value, label }))

export default function MobileMenu() {
  const { group = '' } = useParams()
  const navigate = useNavigate()
  const { preference, setPreference } = useTheme()
  const [aboutOpen, setAboutOpen] = useState(false)
  const menus = advancedMenus
  const menu = menus.find((entry) => entry.key === group)

  if (!menu) return <Navigate to="/" replace />

  const signOut = () => {
    clearToken()
    navigate('/login')
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <PageHeader title={menu.label} />

      {(menu.sections ?? []).map((section) => (
        <PreferencesGroup key={section.title ?? ''} title={section.title}>
          {section.items.map((item) => (
            <Row key={item.to} title={item.label} prefix={<item.icon className="h-5 w-5" />} onClick={() => navigate(item.to)} />
          ))}
        </PreferencesGroup>
      ))}

      {menu.items.length > 0 && (
        <PreferencesGroup>
          {menu.items.map((item) => (
            <Row key={item.to} title={item.label} prefix={<item.icon className="h-5 w-5" />} onClick={() => navigate(item.to)} />
          ))}
        </PreferencesGroup>
      )}

      {menu.key === 'more' && (
        <>
          <PreferencesGroup>
            <Row title="Preferences" prefix={<SlidersHorizontal className="h-5 w-5" />} onClick={() => navigate('/settings')} />
            <Row title="Appearance">
              <Segmented value={preference} onChange={setPreference} options={THEME_OPTIONS} />
            </Row>
          </PreferencesGroup>

          <PreferencesGroup title="Instance">
            <div className="p-2">
              <InstanceMenu tone="sheet" />
            </div>
          </PreferencesGroup>

          <PreferencesGroup>
            <Row title="About Routier" prefix={<Info className="h-5 w-5" />} onClick={() => setAboutOpen(true)} />
            <Row title="Sign Out" prefix={<LogOut className="h-5 w-5" />} onClick={signOut} />
          </PreferencesGroup>

          <AboutDialog open={aboutOpen} onClose={() => setAboutOpen(false)} />
        </>
      )}
    </div>
  )
}
