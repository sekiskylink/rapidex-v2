import React from 'react'
import {
  AppBar,
  Avatar,
  Box,
  Collapse,
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
import TuneRoundedIcon from '@mui/icons-material/TuneRounded'
import BrushRoundedIcon from '@mui/icons-material/BrushRounded'
import WidgetsRoundedIcon from '@mui/icons-material/WidgetsRounded'
import HubRoundedIcon from '@mui/icons-material/HubRounded'
import InfoRoundedIcon from '@mui/icons-material/InfoRounded'
import AccountBalanceWalletRoundedIcon from '@mui/icons-material/AccountBalanceWalletRounded'
import GroupRoundedIcon from '@mui/icons-material/GroupRounded'
import FactCheckRoundedIcon from '@mui/icons-material/FactCheckRounded'
import AdminPanelSettingsRoundedIcon from '@mui/icons-material/AdminPanelSettingsRounded'
import VpnKeyRoundedIcon from '@mui/icons-material/VpnKeyRounded'
import DnsRoundedIcon from '@mui/icons-material/DnsRounded'
import ReceiptLongRoundedIcon from '@mui/icons-material/ReceiptLongRounded'
import LocalShippingRoundedIcon from '@mui/icons-material/LocalShippingRounded'
import WorkOutlineRoundedIcon from '@mui/icons-material/WorkOutlineRounded'
import ScheduleRoundedIcon from '@mui/icons-material/ScheduleRounded'
import VisibilityRoundedIcon from '@mui/icons-material/VisibilityRounded'
import ArticleRoundedIcon from '@mui/icons-material/ArticleRounded'
import ApartmentRoundedIcon from '@mui/icons-material/ApartmentRounded'
import ChevronLeftRoundedIcon from '@mui/icons-material/ChevronLeftRounded'
import ChevronRightRoundedIcon from '@mui/icons-material/ChevronRightRounded'
import ExpandLessRoundedIcon from '@mui/icons-material/ExpandLessRounded'
import ExpandMoreRoundedIcon from '@mui/icons-material/ExpandMoreRounded'
import PaletteRoundedIcon from '@mui/icons-material/PaletteRounded'
import PersonRoundedIcon from '@mui/icons-material/PersonRounded'
import LogoutRoundedIcon from '@mui/icons-material/LogoutRounded'
import { Outlet, useNavigate, useRouter, useRouterState } from '@tanstack/react-router'
import { useSessionPrincipal } from '../auth/hooks'
import { useBootstrapSnapshot } from '../bootstrap/state'
import { buildNavigation, canAccessRoute, type NavigationNode } from '../navigation'
import { getRouteLabel } from '../registry/navigation'
import { clearSession } from '../auth/session'
import { PalettePresetPicker } from '../ui/PalettePresetPicker'
import { useThemePreferences } from '../ui/theme'

const DRAWER_WIDTH = 248
const MINI_DRAWER_WIDTH = 76

function navItemStyles(selected: boolean) {
  return {
    position: 'relative',
    borderRadius: 2,
    mb: 0.5,
    '&::before': {
      content: '""',
      position: 'absolute',
      left: 6,
      top: 8,
      bottom: 8,
      width: 4,
      borderRadius: '0 999px 999px 0',
      backgroundColor: selected ? 'primary.main' : 'transparent',
      transition: 'background-color 120ms ease',
    },
  } as const
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
  const bootstrap = useBootstrapSnapshot()

  const [mobileOpen, setMobileOpen] = React.useState(false)
  const [appearanceOpen, setAppearanceOpen] = React.useState(false)
  const [menuAnchor, setMenuAnchor] = React.useState<null | HTMLElement>(null)
  const [adminExpanded, setAdminExpanded] = React.useState(
    pathname.startsWith('/users') ||
      pathname.startsWith('/roles') ||
      pathname.startsWith('/permissions') ||
      pathname.startsWith('/audit') ||
      pathname.startsWith('/settings'),
  )
  const [sukumadExpanded, setSukumadExpanded] = React.useState(
    pathname.startsWith('/servers') ||
      pathname.startsWith('/requests') ||
      pathname.startsWith('/deliveries') ||
      pathname.startsWith('/jobs') ||
      pathname.startsWith('/scheduler') ||
      pathname.startsWith('/observability') ||
      pathname.startsWith('/documentation'),
  )
  const [settingsExpanded, setSettingsExpanded] = React.useState(pathname.startsWith('/settings'))

  const navCollapsed = !isMobile && prefs.navCollapsed
  const drawerWidth = navCollapsed ? MINI_DRAWER_WIDTH : DRAWER_WIDTH

  const navigation = buildNavigation(principal, {
    labels: prefs.navLabels,
    showAdministration: prefs.showAdministrationMenu,
    showSukumad: prefs.showSukumadMenu,
  })
  const canAccessSettings = canAccessRoute(principal, '/settings/general')
  const displayName = bootstrap.payload?.branding?.applicationDisplayName?.trim() || 'BasePro'

  React.useEffect(() => {
    if (
      pathname.startsWith('/users') ||
      pathname.startsWith('/roles') ||
      pathname.startsWith('/permissions') ||
      pathname.startsWith('/audit') ||
      pathname.startsWith('/settings')
    ) {
      setAdminExpanded(true)
    }
    if (
      pathname.startsWith('/servers') ||
      pathname.startsWith('/requests') ||
      pathname.startsWith('/deliveries') ||
      pathname.startsWith('/jobs') ||
      pathname.startsWith('/scheduler') ||
      pathname.startsWith('/observability') ||
      pathname.startsWith('/documentation') ||
      pathname.startsWith('/orgunits') ||
      pathname.startsWith('/reporters')
    ) {
      setSukumadExpanded(true)
    }
    if (pathname.startsWith('/settings')) {
      setSettingsExpanded(true)
    }
  }, [pathname])

  const navIcons = {
    dashboard: <DashboardRoundedIcon fontSize="small" />,
    settings: <SettingsRoundedIcon fontSize="small" />,
    'settings-general': <TuneRoundedIcon fontSize="small" />,
    'settings-branding': <BrushRoundedIcon fontSize="small" />,
    'settings-modules': <WidgetsRoundedIcon fontSize="small" />,
    'settings-integrations': <HubRoundedIcon fontSize="small" />,
    'settings-about': <InfoRoundedIcon fontSize="small" />,
    sukumad: <AccountBalanceWalletRoundedIcon fontSize="small" />,
    users: <GroupRoundedIcon fontSize="small" />,
    roles: <AdminPanelSettingsRoundedIcon fontSize="small" />,
    permissions: <VpnKeyRoundedIcon fontSize="small" />,
    audit: <FactCheckRoundedIcon fontSize="small" />,
    servers: <DnsRoundedIcon fontSize="small" />,
    requests: <ReceiptLongRoundedIcon fontSize="small" />,
    deliveries: <LocalShippingRoundedIcon fontSize="small" />,
    jobs: <WorkOutlineRoundedIcon fontSize="small" />,
    scheduler: <ScheduleRoundedIcon fontSize="small" />,
    observability: <VisibilityRoundedIcon fontSize="small" />,
    documentation: <ArticleRoundedIcon fontSize="small" />,
    orgunits: <ApartmentRoundedIcon fontSize="small" />,
    reporters: <PersonRoundedIcon fontSize="small" />,
  }

  const closeMenus = () => {
    setMenuAnchor(null)
  }

  function renderAdministrationItem(item: NavigationNode) {
    const icon = navIcons[item.key as keyof typeof navIcons] ?? <SettingsRoundedIcon fontSize="small" />

    if ('children' in item) {
      const selected = pathname.startsWith('/settings')
      const button = (
        <ListItemButton
          key={item.key}
          selected={selected}
          aria-label={item.label}
          aria-expanded={settingsExpanded}
          onClick={() => setSettingsExpanded((current) => !current)}
          sx={{
            ...navItemStyles(selected),
            justifyContent: navCollapsed ? 'center' : 'flex-start',
            px: navCollapsed ? 1 : 2.75,
          }}
        >
          <ListItemIcon sx={{ minWidth: navCollapsed ? 0 : 34, justifyContent: 'center' }}>{icon}</ListItemIcon>
          {!navCollapsed ? <ListItemText primary={item.label} /> : null}
          {!navCollapsed ? (settingsExpanded ? <ExpandLessRoundedIcon fontSize="small" /> : <ExpandMoreRoundedIcon fontSize="small" />) : null}
        </ListItemButton>
      )

      const children = item.children.map((child) => {
        if ('children' in child) {
          return null
        }
        const childSelected = pathname.startsWith(child.path)
        if (navCollapsed) {
          return (
            <ListItemButton
              key={child.key}
              selected={childSelected}
              onClick={() => {
                void navigate({ to: child.path })
                setMobileOpen(false)
              }}
              sx={{
                ...navItemStyles(childSelected),
                justifyContent: 'center',
                px: 1,
              }}
            >
              <ListItemIcon sx={{ minWidth: 0, justifyContent: 'center' }}>
                {navIcons[child.key as keyof typeof navIcons] ?? <SettingsRoundedIcon fontSize="small" />}
              </ListItemIcon>
            </ListItemButton>
          )
        }
        return (
          <ListItemButton
            key={child.key}
            selected={childSelected}
            onClick={() => {
              void navigate({ to: child.path })
              setMobileOpen(false)
            }}
            sx={{
              ...navItemStyles(childSelected),
              justifyContent: 'flex-start',
              px: 3.75,
            }}
          >
            <ListItemIcon sx={{ minWidth: 34, justifyContent: 'center' }}>
              {navIcons[child.key as keyof typeof navIcons] ?? <SettingsRoundedIcon fontSize="small" />}
            </ListItemIcon>
            <ListItemText primary={child.label} />
          </ListItemButton>
        )
      })

      return (
        <React.Fragment key={item.key}>
          {button}
          {navCollapsed ? (settingsExpanded ? children : null) : <Collapse in={settingsExpanded} unmountOnExit>{children}</Collapse>}
        </React.Fragment>
      )
    }

    const selected = pathname.startsWith(item.path)
    return (
      <ListItemButton
        key={item.key}
        selected={selected}
        onClick={() => {
          void navigate({ to: item.path })
          setMobileOpen(false)
        }}
        sx={{
          ...navItemStyles(selected),
          justifyContent: navCollapsed ? 'center' : 'flex-start',
          px: navCollapsed ? 1 : 2.75,
        }}
      >
        <ListItemIcon sx={{ minWidth: navCollapsed ? 0 : 34, justifyContent: 'center' }}>{icon}</ListItemIcon>
        {!navCollapsed ? <ListItemText primary={item.label} /> : null}
      </ListItemButton>
    )
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
            {displayName}
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
        {navigation.topLevel.map((item) => (
          <ListItemButton
            key={item.key}
            selected={pathname.startsWith(item.path)}
            onClick={() => {
              void navigate({ to: item.path })
              setMobileOpen(false)
            }}
            sx={{
              ...navItemStyles(pathname.startsWith(item.path)),
              justifyContent: navCollapsed ? 'center' : 'flex-start',
              px: navCollapsed ? 1 : 1.75,
            }}
          >
            <ListItemIcon sx={{ minWidth: navCollapsed ? 0 : 34, justifyContent: 'center' }}>
              {navIcons[item.key as keyof typeof navIcons]}
            </ListItemIcon>
            {!navCollapsed ? <ListItemText primary={item.label} /> : null}
          </ListItemButton>
        ))}
        {navigation.sukumad.visible ? (
          <>
            <ListItemButton
              aria-label="Toggle Sukumad menu"
              aria-expanded={sukumadExpanded}
              onClick={() => setSukumadExpanded((current) => !current)}
              sx={{
                ...navItemStyles(false),
                justifyContent: navCollapsed ? 'center' : 'flex-start',
                px: navCollapsed ? 1 : 1.75,
              }}
            >
              <ListItemIcon sx={{ minWidth: navCollapsed ? 0 : 34, justifyContent: 'center' }}>
                {navIcons.sukumad}
              </ListItemIcon>
              {!navCollapsed ? <ListItemText primary={navigation.sukumad.label} /> : null}
              {!navCollapsed ? (sukumadExpanded ? <ExpandLessRoundedIcon fontSize="small" /> : <ExpandMoreRoundedIcon fontSize="small" />) : null}
            </ListItemButton>
            {navCollapsed ? (
              sukumadExpanded
                ? navigation.sukumad.children.map((item) => {
                    if ('children' in item) {
                      return null
                    }
                    return (
                      <ListItemButton
                        key={item.key}
                        selected={pathname.startsWith(item.path)}
                        onClick={() => {
                          void navigate({ to: item.path })
                          setMobileOpen(false)
                        }}
                        sx={{
                          ...navItemStyles(pathname.startsWith(item.path)),
                          justifyContent: 'center',
                          px: 1,
                        }}
                      >
                        <ListItemIcon sx={{ minWidth: 0, justifyContent: 'center' }}>
                          {navIcons[item.key as keyof typeof navIcons]}
                        </ListItemIcon>
                      </ListItemButton>
                    )
                  })
                : null
            ) : (
              <Collapse in={sukumadExpanded} unmountOnExit>
                {navigation.sukumad.children.map((item) => {
                  if ('children' in item) {
                    return null
                  }
                  return (
                    <ListItemButton
                      key={item.key}
                      selected={pathname.startsWith(item.path)}
                      onClick={() => {
                        void navigate({ to: item.path })
                        setMobileOpen(false)
                      }}
                      sx={{
                        ...navItemStyles(pathname.startsWith(item.path)),
                        justifyContent: 'flex-start',
                        px: 2.75,
                      }}
                    >
                      <ListItemIcon sx={{ minWidth: 34, justifyContent: 'center' }}>
                        {navIcons[item.key as keyof typeof navIcons]}
                      </ListItemIcon>
                      <ListItemText primary={item.label} />
                    </ListItemButton>
                  )
                })}
              </Collapse>
            )}
          </>
        ) : null}
        {navigation.administration.visible ? (
          <>
            <ListItemButton
              aria-label="Toggle Administration menu"
              aria-expanded={adminExpanded}
              onClick={() => setAdminExpanded((current) => !current)}
              sx={{
                ...navItemStyles(false),
                justifyContent: navCollapsed ? 'center' : 'flex-start',
                px: navCollapsed ? 1 : 1.75,
              }}
            >
              <ListItemIcon sx={{ minWidth: navCollapsed ? 0 : 34, justifyContent: 'center' }}>
                <AdminPanelSettingsRoundedIcon fontSize="small" />
              </ListItemIcon>
              {!navCollapsed ? <ListItemText primary={navigation.administration.label} /> : null}
              {!navCollapsed ? (adminExpanded ? <ExpandLessRoundedIcon fontSize="small" /> : <ExpandMoreRoundedIcon fontSize="small" />) : null}
            </ListItemButton>
            {navCollapsed ? (
              adminExpanded
                ? navigation.administration.children.map((item) => renderAdministrationItem(item))
                : null
            ) : (
              <Collapse in={adminExpanded} unmountOnExit>
                {navigation.administration.children.map((item) => renderAdministrationItem(item))}
              </Collapse>
            )}
          </>
        ) : null}
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
              {displayName} Shell
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
            {getRouteLabel(pathname)}
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
            {canAccessSettings ? (
              <MenuItem
                onClick={() => {
                  closeMenus()
                  void navigate({ to: '/settings/general' })
                }}
              >
                <ListItemIcon>
                  <SettingsRoundedIcon fontSize="small" />
                </ListItemIcon>
                Settings
              </MenuItem>
            ) : null}
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
            {displayName} Desktop v0.1.0
          </Typography>
        </Box>
      </Box>

      <PalettePresetPicker open={appearanceOpen} onClose={() => setAppearanceOpen(false)} />
    </Box>
  )
}
