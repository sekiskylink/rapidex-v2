import React from 'react'
import {
  AppBar,
  Avatar,
  Box,
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
import { Outlet, useNavigate, useRouterState } from '@tanstack/react-router'
import { useAuth } from '../auth/AuthProvider'
import { appName } from '../lib/env'
import { buildNavigation } from '../navigation'
import {
  AdminPanelSettingsRoundedIcon,
  ChevronLeftRoundedIcon,
  ChevronRightRoundedIcon,
  CloseIcon,
  DashboardRoundedIcon,
  FactCheckRoundedIcon,
  GroupRoundedIcon,
  LogoutRoundedArrowIcon,
  MenuIcon,
  PaletteRoundedIcon,
  SettingsRoundedIcon,
  VpnKeyRoundedIcon,
} from '../ui/icons'
import { PalettePresetPicker } from '../ui/theme/PalettePresetPicker'
import { useUiPreferences } from '../ui/theme/UiPreferencesProvider'

const drawerWidth = 260
const miniDrawerWidth = 80

function sectionTitle(pathname: string) {
  if (pathname.startsWith('/users')) {
    return 'Users'
  }
  if (pathname.startsWith('/roles')) {
    return 'Roles'
  }
  if (pathname.startsWith('/permissions')) {
    return 'Permissions'
  }
  if (pathname.startsWith('/audit')) {
    return 'Audit Log'
  }
  if (pathname.startsWith('/settings')) {
    return 'Settings'
  }
  if (pathname.startsWith('/dashboard')) {
    return 'Dashboard'
  }
  return 'BasePro'
}

export function AppShell() {
  const navigate = useNavigate()
  const { logout, user } = useAuth()
  const { prefs, setCollapseNavByDefault } = useUiPreferences()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const [collapsed, setCollapsed] = React.useState(prefs.collapseNavByDefault)
  const [mobileOpen, setMobileOpen] = React.useState(false)
  const [appearanceOpen, setAppearanceOpen] = React.useState(false)
  const [menuAnchor, setMenuAnchor] = React.useState<null | HTMLElement>(null)
  const theme = useTheme()
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'), { noSsr: true })
  const firstNavItemRef = React.useRef<HTMLDivElement | null>(null)

  React.useEffect(() => {
    setCollapsed(prefs.collapseNavByDefault)
  }, [prefs.collapseNavByDefault])

  React.useEffect(() => {
    if (isMobile) {
      setMobileOpen(false)
    }
  }, [isMobile, pathname])

  React.useEffect(() => {
    if (isMobile && mobileOpen) {
      firstNavItemRef.current?.focus()
    }
  }, [isMobile, mobileOpen])

  const navigation = buildNavigation(user)
  const navIcons = {
    dashboard: <DashboardRoundedIcon fontSize="small" />,
    settings: <SettingsRoundedIcon fontSize="small" />,
    users: <GroupRoundedIcon fontSize="small" />,
    roles: <AdminPanelSettingsRoundedIcon fontSize="small" />,
    permissions: <VpnKeyRoundedIcon fontSize="small" />,
    audit: <FactCheckRoundedIcon fontSize="small" />,
  }
  const activeDrawerWidth = collapsed ? miniDrawerWidth : drawerWidth

  const handleDesktopDrawerToggle = () => {
    const next = !collapsed
    setCollapsed(next)
    setCollapseNavByDefault(next)
  }

  const handleMobileDrawerOpen = () => {
    setMobileOpen(true)
  }

  const handleMobileDrawerClose = () => {
    setMobileOpen(false)
  }

  const handleNavItemClick = (path: string) => {
    void navigate({ to: path })
    if (isMobile) {
      setMobileOpen(false)
    }
  }

  const closeMenus = () => {
    setMenuAnchor(null)
  }

  const drawer = (
    <Box sx={{ display: 'flex', height: '100%', flexDirection: 'column' }}>
      <Toolbar sx={{ justifyContent: collapsed ? 'center' : 'space-between', px: 1.5 }}>
        {!collapsed ? (
          <Typography variant="subtitle1" component="div" sx={{ fontWeight: 600 }}>
            {appName}
          </Typography>
        ) : null}
        {!isMobile ? (
          <IconButton
            aria-label={collapsed ? 'Expand navigation' : 'Collapse navigation'}
            edge="end"
            onClick={handleDesktopDrawerToggle}
          >
            {collapsed ? <ChevronRightRoundedIcon /> : <ChevronLeftRoundedIcon />}
          </IconButton>
        ) : null}
      </Toolbar>
      <Divider />
      <List sx={{ px: 1, py: 1.5 }}>
        {navigation.topLevel.map((item, index) => {
          const selected = pathname.startsWith(item.path)
          const button = (
            <ListItemButton
              key={item.key}
              ref={index === 0 ? firstNavItemRef : undefined}
              selected={selected}
              onClick={() => handleNavItemClick(item.path)}
              aria-label={item.label}
              sx={{
                minHeight: 46,
                mb: 0.5,
                justifyContent: collapsed && !isMobile ? 'center' : 'flex-start',
                borderRadius: 1.5,
              }}
            >
              <ListItemIcon sx={{ minWidth: collapsed && !isMobile ? 'auto' : 36 }}>
                {navIcons[item.key as keyof typeof navIcons]}
              </ListItemIcon>
              <ListItemText
                primary={item.label}
                primaryTypographyProps={{
                  noWrap: true,
                  fontWeight: selected ? 600 : 500,
                }}
                sx={{
                  opacity: collapsed && !isMobile ? 0 : 1,
                  transition: theme.transitions.create('opacity', {
                    duration: theme.transitions.duration.shortest,
                  }),
                }}
              />
            </ListItemButton>
          )

          if (collapsed && !isMobile) {
            return (
              <Tooltip key={item.path} title={item.label} placement="right">
                {button}
              </Tooltip>
            )
          }

          return button
        })}
        {navigation.administration.visible ? (
          <>
            {!collapsed || isMobile ? (
              <Typography variant="overline" color="text.secondary" sx={{ px: 1.5, pt: 1.25, pb: 0.5, display: 'block' }}>
                Administration
              </Typography>
            ) : null}
            {navigation.administration.children.map((item) => {
              const selected = pathname.startsWith(item.path)
              const button = (
                <ListItemButton
                  key={item.key}
                  selected={selected}
                  onClick={() => handleNavItemClick(item.path)}
                  aria-label={item.label}
                  sx={{
                    minHeight: 46,
                    mb: 0.5,
                    justifyContent: collapsed && !isMobile ? 'center' : 'flex-start',
                    borderRadius: 1.5,
                    pl: collapsed && !isMobile ? 1.5 : 2.5,
                  }}
                >
                  <ListItemIcon sx={{ minWidth: collapsed && !isMobile ? 'auto' : 36 }}>
                    {navIcons[item.key as keyof typeof navIcons]}
                  </ListItemIcon>
                  <ListItemText
                    primary={item.label}
                    primaryTypographyProps={{
                      noWrap: true,
                      fontWeight: selected ? 600 : 500,
                    }}
                    sx={{
                      opacity: collapsed && !isMobile ? 0 : 1,
                      transition: theme.transitions.create('opacity', {
                        duration: theme.transitions.duration.shortest,
                      }),
                    }}
                  />
                </ListItemButton>
              )

              if (collapsed && !isMobile) {
                return (
                  <Tooltip key={item.key} title={item.label} placement="right">
                    {button}
                  </Tooltip>
                )
              }

              return button
            })}
          </>
        ) : null}
      </List>
    </Box>
  )

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }} data-testid="app-shell">
      <AppBar
        position="fixed"
        sx={{
          zIndex: (muiTheme) => muiTheme.zIndex.drawer + 1,
          transition: theme.transitions.create(['width', 'margin-left'], {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.enteringScreen,
          }),
          ...(isMobile
            ? undefined
            : {
                marginLeft: `${activeDrawerWidth}px`,
                width: `calc(100% - ${activeDrawerWidth}px)`,
              }),
        }}
      >
        <Toolbar>
          <IconButton
            color="inherit"
            edge="start"
            aria-label={isMobile ? 'Open navigation menu' : collapsed ? 'Expand navigation' : 'Collapse navigation'}
            onClick={isMobile ? handleMobileDrawerOpen : handleDesktopDrawerToggle}
            sx={{ mr: 1.5 }}
          >
            {isMobile ? <MenuIcon /> : collapsed ? <ChevronRightRoundedIcon /> : <ChevronLeftRoundedIcon />}
          </IconButton>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }} noWrap>
            {sectionTitle(pathname)}
          </Typography>
          <Typography variant="body2" sx={{ display: { xs: 'none', sm: 'block' }, mr: 1.25, opacity: 0.95 }}>
            {user?.username ?? 'User'}
          </Typography>
          <Tooltip title="Open user menu">
            <IconButton color="inherit" onClick={(event) => setMenuAnchor(event.currentTarget)} aria-label="Open user menu">
              <Avatar sx={{ width: 32, height: 32 }}>{(user?.username ?? 'U').slice(0, 1).toUpperCase()}</Avatar>
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
            <MenuItem
              onClick={() => {
                closeMenus()
                void logout()
              }}
            >
              <ListItemIcon>
                <LogoutRoundedArrowIcon fontSize="small" />
              </ListItemIcon>
              Logout
            </MenuItem>
          </Menu>
        </Toolbar>
      </AppBar>

      <Box component="nav" aria-label="Primary navigation">
        {isMobile ? (
          <Drawer
            variant="temporary"
            open={mobileOpen}
            onClose={handleMobileDrawerClose}
            ModalProps={{ keepMounted: true }}
            sx={{
              display: { xs: 'block', md: 'none' },
              '& .MuiDrawer-paper': {
                width: drawerWidth,
                boxSizing: 'border-box',
              },
            }}
          >
            <Box sx={{ display: 'flex', justifyContent: 'flex-end', px: 1, pt: 1 }}>
              <IconButton aria-label="Close navigation menu" onClick={handleMobileDrawerClose}>
                <CloseIcon />
              </IconButton>
            </Box>
            {drawer}
          </Drawer>
        ) : (
          <Drawer
            variant="permanent"
            sx={{
              width: activeDrawerWidth,
              flexShrink: 0,
              '& .MuiDrawer-paper': {
                width: activeDrawerWidth,
                boxSizing: 'border-box',
                overflowX: 'hidden',
                transition: theme.transitions.create('width', {
                  easing: theme.transitions.easing.sharp,
                  duration: theme.transitions.duration.enteringScreen,
                }),
              },
            }}
          >
            {drawer}
          </Drawer>
        )}
      </Box>

      <Box
        component="main"
        sx={{
          transition: theme.transitions.create(['width', 'margin-left'], {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.enteringScreen,
          }),
          ...(isMobile
            ? undefined
            : {
                marginLeft: `${activeDrawerWidth}px`,
                width: `calc(100% - ${activeDrawerWidth}px)`,
              }),
        }}
      >
        <Toolbar />
        <Box sx={{ display: 'flex', minHeight: 'calc(100vh - 64px)', flexDirection: 'column' }}>
          <Box sx={{ flexGrow: 1, p: { xs: 2, sm: 3 } }}>
            <Outlet />
          </Box>
          {prefs.showFooter ? (
            <Box
              component="footer"
              sx={{
                borderTop: 1,
                borderColor: 'divider',
                px: { xs: 2, sm: 3 },
                py: 1.25,
                bgcolor: 'background.paper',
              }}
            >
              <Typography variant="body2" color="text.secondary">
                {appName} v0.1.0
              </Typography>
            </Box>
          ) : null}
        </Box>
      </Box>
      <PalettePresetPicker open={appearanceOpen} onClose={() => setAppearanceOpen(false)} />
    </Box>
  )
}
