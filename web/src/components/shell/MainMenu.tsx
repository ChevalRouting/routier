import { MenuButton, useTheme, type MenuItem, type ThemePreference } from 'cheval-ui'
import { Info, LogOut, Menu, Monitor, Moon, SlidersHorizontal, Sun } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { AboutDialog } from '@/components/shell/AboutDialog'
import { clearToken } from '@/lib/utils'

interface AppearanceOption {
  value: ThemePreference
  label: string
  icon: MenuItem['icon']
}

export const APPEARANCE_OPTIONS: AppearanceOption[] = [
  { value: 'system', label: 'Follow System', icon: Monitor },
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'dark', label: 'Dark', icon: Moon },
]

interface MainMenuProps {
  className?: string
}

export function MainMenu({ className }: MainMenuProps) {
  const navigate = useNavigate()
  const { preference, setPreference } = useTheme()
  const [aboutOpen, setAboutOpen] = useState(false)

  const signOut = () => {
    clearToken()
    navigate('/login')
  }

  const sections: MenuItem[][] = [
    APPEARANCE_OPTIONS.map((option) => ({
      key: `appearance-${option.value}`,
      label: option.label,
      icon: option.icon,
      checked: preference === option.value,
      onSelect: () => setPreference(option.value),
    })),
    [
      { key: 'preferences', label: 'Preferences', icon: SlidersHorizontal, onSelect: () => navigate('/settings') },
      { key: 'about', label: 'About Routier', icon: Info, onSelect: () => setAboutOpen(true) },
    ],
    [{ key: 'sign-out', label: 'Sign Out', icon: LogOut, onSelect: signOut }],
  ]

  return (
    <>
      <MenuButton label="Main Menu" sections={sections} align="start" className={className}>
        <Menu className="h-4 w-4" />
      </MenuButton>
      <AboutDialog open={aboutOpen} onClose={() => setAboutOpen(false)} />
    </>
  )
}
