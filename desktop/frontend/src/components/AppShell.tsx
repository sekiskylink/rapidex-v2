import React from 'react'
import {
  AppBar,
  Avatar,
  Box,
  Chip,
  Divider,
  Drawer,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Toolbar,
  Tooltip,
  Typography,
  useMediaQuery,
  useTheme,
} from '@mui/material'
import { alpha } from '@mui/material/styles'
import MenuIcon from '@mui/icons-material/Menu'
import DashboardRoundedIcon from '@mui/icons-material/DashboardRounded'
import SettingsRoundedIcon from '@mui/icons-material/SettingsRounded'
import GroupRoundedIcon from '@mui/icons-material/GroupRounded'
import FactCheckRoundedIcon from '@mui/icons-material/FactCheckRounded'
import ChevronLeftRoundedIcon from '@mui/icons-material/ChevronLeftRounded'
import ChevronRightRoundedIcon from '@mui/icons-material/ChevronRightRounded'
import PaletteRoundedIcon from '@mui/icons-material/PaletteRounded'
import LogoutRoundedIcon from '@mui/icons-material/LogoutRounded'
import { Outlet, useNavigate, useRouter, useRouterState } from '@tanstack/react-router'
import { useSessionPrincipal } from '../auth/hooks'
import { clearSession } from '../auth/session'
import { PalettePresetPicker } from '../ui/PalettePresetPicker'
import { useThemePreferences } from '../ui/theme'

const DRAWER_WIDTH = 248
const MINI_DRAWER_WIDTH = 76

function sectionTitle(pathname: string) {
  if (pathname.startsWith('/users')) {
    return 'Users'
  }
  if (pathname.startsWith('/audit')) {
    return 'Audit'
  }
  if (pathname.startsWith('/settings')) {
    return 'Settings'
  }
  if (pathname.startsWith('/dashboard')) {
    return 'Dashboard'
  }
  return 'BasePro'
}

function normalizeBaseUrl(baseUrl: string) {
  const trimmed = baseUrl.trim().replace(/\/+$/, '')
  return trimmed.endsWith('/api/v1') ? trimmed.slice(0, -'/api/v1'.length) : trimmed
}

export function AppShell() {
  const theme = useTheme()
  const navigate = useNavigate()
  const router = useRouter()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const isMobile = useMediaQuery(theme.breakpoints.down('md'))
  const { prefs, setNavCollapsed } = useThemePreferences()
  const principal = useSessionPrincipal()

  const [mobileOpen, setMobileOpen] = React.useState(false)
  const [appearanceOpen, setAppearanceOpen] = React.useState(false)
  const [menuAnchor, setMenuAnchor] = React.useState<null | HTMLElement>(null)

  const navCollapsed = !isMobile && prefs.navCollapsed
  const drawerWidth = navCollapsed ? MINI_DRAWER_WIDTH : DRAWER_WIDTH

  const navItems = [
    {
      label: 'Dashboard',
      path: '/dashboard',
      icon: <DashboardRoundedIcon fontSize="small" />,
      active: pathname.startsWith('/dashboard'),
      disabled: false,
    },
    {
      label: 'Settings',
      path: '/settings',
      icon: <SettingsRoundedIcon fontSize="small" />,
      active: pathname.startsWith('/settings'),
      disabled: false,
    },
  ]
  if (principal && (principal.permissions.includes('users.read') || principal.permissions.includes('users.write'))) {
    navItems.push({
      label: 'Users',
      path: '/users',
      icon: <GroupRoundedIcon fontSize="small" />,
      active: pathname.startsWith('/users'),
      disabled: false,
    })
  }
  if (principal && principal.permissions.includes('audit.read')) {
    navItems.push({
      label: 'Audit',
      path: '/audit',
      icon: <FactCheckRoundedIcon fontSize="small" />,
      active: pathname.startsWith('/audit'),
      disabled: false,
    })
  }

  const closeMenus = () => {
    setMenuAnchor(null)
  }

  const handleLogout = async () => {
    closeMenus()

    const settings = await router.options.context.settingsStore.loadSettings()
    const refreshToken = settings.refreshToken?.trim()
    const baseUrl = normalizeBaseUrl(settings.apiBaseUrl)

    if (refreshToken && baseUrl) {
      try {
        await fetch(`${baseUrl}/api/v1/auth/logout`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refreshToken }),
        })
      } catch {
        // Best effort server logout.
      }
    }

    await clearSession()
    await navigate({ to: '/login', replace: true })
  }

  const drawerContent = (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Toolbar
        sx={{
          justifyContent: navCollapsed ? 'center' : 'space-between',
          px: 1.5,
          bgcolor: alpha(theme.palette.primary.main, theme.palette.mode === 'light' ? 0.9 : 0.86),
          color: 'primary.contrastText',
        }}
      >
        {!navCollapsed ? (
          <Typography variant="h6" noWrap>
            BasePro
          </Typography>
        ) : null}
        {!isMobile ? (
          <Tooltip title={navCollapsed ? 'Expand menu' : 'Collapse menu'}>
            <IconButton onClick={() => void setNavCollapsed(!prefs.navCollapsed)}>
              {navCollapsed ? <ChevronRightRoundedIcon /> : <ChevronLeftRoundedIcon />}
            </IconButton>
          </Tooltip>
        ) : null}
      </Toolbar>

      <Divider />

      <List sx={{ px: 1, py: 1 }}>
        {navItems.map((item) => (
          <ListItemButton
            key={item.label}
            selected={item.active}
            disabled={item.disabled}
            onClick={() => {
              if (item.disabled) {
                return
              }
              void navigate({ to: item.path })
              setMobileOpen(false)
            }}
            sx={{
              borderRadius: 2,
              mb: 0.5,
              justifyContent: navCollapsed ? 'center' : 'flex-start',
              px: navCollapsed ? 1 : 1.5,
            }}
          >
            <ListItemIcon sx={{ minWidth: navCollapsed ? 0 : 34, justifyContent: 'center' }}>
              {item.icon}
            </ListItemIcon>
            {!navCollapsed ? (
              <ListItemText
                primary={item.label}
                secondary={item.disabled ? 'Coming soon' : undefined}
                slotProps={{
                  secondary: {
                    sx: {
                      color: 'text.disabled',
                    },
                  },
                }}
              />
            ) : null}
            {!navCollapsed && item.disabled ? <Chip label="Soon" size="small" /> : null}
          </ListItemButton>
        ))}
      </List>
      <Box sx={{ mt: 'auto', p: 1.25 }}>
        <Box
          sx={{
            borderRadius: 2,
            px: navCollapsed ? 0.5 : 1.25,
            py: 0.75,
            textAlign: navCollapsed ? 'center' : 'left',
            bgcolor: alpha(theme.palette.primary.main, theme.palette.mode === 'light' ? 0.12 : 0.24),
            color: 'primary.main',
          }}
        >
          {!navCollapsed ? (
            <Typography variant="caption" fontWeight={700}>
              BasePro Shell
            </Typography>
          ) : (
            <Typography variant="caption" fontWeight={700}>
              BP
            </Typography>
          )}
        </Box>
      </Box>
    </Box>
  )

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh', bgcolor: 'background.default' }}>
      <AppBar
        position="fixed"
        color="primary"
        sx={{
          ml: { md: `${drawerWidth}px` },
          width: { md: `calc(100% - ${drawerWidth}px)` },
          backgroundImage: 'none',
          bgcolor: 'primary.main',
          color: 'primary.contrastText',
          borderColor: theme.palette.mode === 'light' ? 'rgba(255,255,255,0.24)' : 'rgba(0,0,0,0.28)',
        }}
      >
        <Toolbar>
          {isMobile ? (
            <IconButton edge="start" color="inherit" onClick={() => setMobileOpen(true)} sx={{ mr: 1.5 }}>
              <MenuIcon />
            </IconButton>
          ) : null}
          <Typography variant="h6" sx={{ flexGrow: 1 }}>
            {sectionTitle(pathname)}
          </Typography>
          <Tooltip title="Open user menu">
            <IconButton onClick={(event) => setMenuAnchor(event.currentTarget)}>
              <Avatar sx={{ width: 32, height: 32 }}>
                {(principal?.username ?? 'U').slice(0, 1).toUpperCase()}
              </Avatar>
            </IconButton>
          </Tooltip>
          <Menu
            anchorEl={menuAnchor}
            open={Boolean(menuAnchor)}
            onClose={closeMenus}
            anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
            transformOrigin={{ horizontal: 'right', vertical: 'top' }}
          >
            <MenuItem
              onClick={() => {
                closeMenus()
                void navigate({ to: '/settings' })
              }}
            >
              <ListItemIcon>
                <SettingsRoundedIcon fontSize="small" />
              </ListItemIcon>
              Settings
            </MenuItem>
            <MenuItem
              onClick={() => {
                closeMenus()
                setAppearanceOpen(true)
              }}
            >
              <ListItemIcon>
                <PaletteRoundedIcon fontSize="small" />
              </ListItemIcon>
              Appearance
            </MenuItem>
            <Divider />
            <MenuItem onClick={() => void handleLogout()}>
              <ListItemIcon>
                <LogoutRoundedIcon fontSize="small" />
              </ListItemIcon>
              Logout
            </MenuItem>
          </Menu>
        </Toolbar>
      </AppBar>

      <Box component="nav" sx={{ width: { md: drawerWidth }, flexShrink: { md: 0 } }}>
        <Drawer
          variant="temporary"
          open={mobileOpen}
          onClose={() => setMobileOpen(false)}
          ModalProps={{ keepMounted: true }}
          sx={{
            display: { xs: 'block', md: 'none' },
            '& .MuiDrawer-paper': { width: DRAWER_WIDTH, boxSizing: 'border-box' },
          }}
        >
          {drawerContent}
        </Drawer>
        <Drawer
          variant="permanent"
          open
          sx={{
            display: { xs: 'none', md: 'block' },
            '& .MuiDrawer-paper': {
              width: drawerWidth,
              overflowX: 'hidden',
              transition: theme.transitions.create('width', {
                duration: theme.transitions.duration.enteringScreen,
              }),
              boxSizing: 'border-box',
            },
          }}
        >
          {drawerContent}
        </Drawer>
      </Box>

      <Box component="main" sx={{ display: 'flex', flexDirection: 'column', flexGrow: 1, minHeight: '100vh' }}>
        <Toolbar />
        <Box sx={{ flexGrow: 1, px: { xs: 2, md: 4 }, py: 3 }}>
          <Box sx={{ maxWidth: 1280, mx: 'auto' }}>
            <Outlet />
          </Box>
        </Box>
        <Box
          component="footer"
          sx={{
            px: { xs: 2, md: 4 },
            py: 1.5,
            borderTop: '1px solid',
            borderColor: 'divider',
            bgcolor: 'background.paper',
          }}
        >
          <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 1280, mx: 'auto' }}>
            BasePro Desktop v0.1.0
          </Typography>
        </Box>
      </Box>

      <PalettePresetPicker open={appearanceOpen} onClose={() => setAppearanceOpen(false)} />
    </Box>
  )
}
