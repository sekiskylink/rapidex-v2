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
import { Outlet, useNavigate, useRouterState } from '@tanstack/react-router'
import { useAuth } from '../auth/AuthProvider'
import { useBootstrapSnapshot } from '../bootstrap/state'
import { appName } from '../lib/env'
import { buildNavigation, canAccessRoute } from '../navigation'
import { getRouteLabel } from '../registry/navigation'
import {
  AdminPanelSettingsRoundedIcon,
  ChevronLeftRoundedIcon,
  ChevronRightRoundedIcon,
  CloseIcon,
  DashboardRoundedIcon,
  ExpandLessRoundedIcon,
  ExpandMoreRoundedIcon,
  FactCheckRoundedIcon,
  GroupRoundedIcon,
  AccountBalanceWalletRoundedIcon,
  LogoutRoundedArrowIcon,
  MenuIcon,
  PaletteRoundedIcon,
  SettingsRoundedIcon,
  BadgeRoundedIcon,
  EventAvailableRoundedIcon,
  VisibilityRoundedIcon,
  VpnKeyRoundedIcon,
  DnsRoundedIcon,
  ReceiptLongRoundedIcon,
  LocalShippingRoundedIcon,
  WorkOutlineRoundedIcon,
  ArticleRoundedIcon,
  ApartmentRoundedIcon,
  PersonRoundedIcon,
} from '../ui/icons'
import { PalettePresetPicker } from '../ui/theme/PalettePresetPicker'
import { useUiPreferences } from '../ui/theme/UiPreferencesProvider'

const drawerWidth = 260
const miniDrawerWidth = 80

function navItemStyles(selected: boolean) {
  return {
    position: 'relative',
    minHeight: 46,
    mb: 0.5,
    borderRadius: 1.5,
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

export function AppShell() {
  const navigate = useNavigate()
  const { logout, user } = useAuth()
  const bootstrap = useBootstrapSnapshot()
  const { prefs, setCollapseNavByDefault } = useUiPreferences()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const [collapsed, setCollapsed] = React.useState(prefs.collapseNavByDefault)
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
      pathname.startsWith('/documentation') ||
      pathname.startsWith('/orgunits') ||
      pathname.startsWith('/reporters'),
  )
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
  }, [pathname])

  const navigation = buildNavigation(user, {
    labels: prefs.navLabels,
    showAdministration: prefs.showAdministrationMenu,
    showSukumad: prefs.showSukumadMenu,
  })
  const canAccessSettings = canAccessRoute('/settings', user)
  const displayName = bootstrap.payload?.branding?.applicationDisplayName?.trim() || appName
  const navIcons = {
    dashboard: <DashboardRoundedIcon fontSize="small" />,
    settings: <SettingsRoundedIcon fontSize="small" />,
    sukumad: <AccountBalanceWalletRoundedIcon fontSize="small" />,
    users: <GroupRoundedIcon fontSize="small" />,
    roles: <AdminPanelSettingsRoundedIcon fontSize="small" />,
    permissions: <VpnKeyRoundedIcon fontSize="small" />,
    audit: <FactCheckRoundedIcon fontSize="small" />,
    servers: <DnsRoundedIcon fontSize="small" />,
    requests: <ReceiptLongRoundedIcon fontSize="small" />,
    deliveries: <LocalShippingRoundedIcon fontSize="small" />,
    jobs: <WorkOutlineRoundedIcon fontSize="small" />,
    scheduler: <EventAvailableRoundedIcon fontSize="small" />,
    observability: <VisibilityRoundedIcon fontSize="small" />,
    documentation: <ArticleRoundedIcon fontSize="small" />,
    orgunits: <ApartmentRoundedIcon fontSize="small" />,
    reporters: <PersonRoundedIcon fontSize="small" />,
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
            {displayName}
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
                ...navItemStyles(selected),
                justifyContent: collapsed && !isMobile ? 'center' : 'flex-start',
                pl: collapsed && !isMobile ? 1.5 : 2.25,
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
        {navigation.sukumad.visible ? (
          <>
            <ListItemButton
              aria-label="Toggle Sukumad menu"
              aria-expanded={sukumadExpanded}
              onClick={() => setSukumadExpanded((current) => !current)}
              sx={{
                ...navItemStyles(false),
                justifyContent: collapsed && !isMobile ? 'center' : 'flex-start',
                pl: collapsed && !isMobile ? 1.5 : 2.25,
              }}
            >
              <ListItemIcon sx={{ minWidth: collapsed && !isMobile ? 'auto' : 36 }}>
                {navIcons.sukumad}
              </ListItemIcon>
              <ListItemText
                primary={navigation.sukumad.label}
                primaryTypographyProps={{
                  noWrap: true,
                  fontWeight: 600,
                }}
                sx={{
                  opacity: collapsed && !isMobile ? 0 : 1,
                  transition: theme.transitions.create('opacity', {
                    duration: theme.transitions.duration.shortest,
                  }),
                }}
              />
              {!collapsed || isMobile ? (sukumadExpanded ? <ExpandLessRoundedIcon fontSize="small" /> : <ExpandMoreRoundedIcon fontSize="small" />) : null}
            </ListItemButton>
            {collapsed && !isMobile ? (
              sukumadExpanded
                ? navigation.sukumad.children.map((item) => {
                    const selected = pathname.startsWith(item.path)
                    return (
                      <Tooltip key={item.key} title={item.label} placement="right">
                        <ListItemButton
                          selected={selected}
                          onClick={() => handleNavItemClick(item.path)}
                          aria-label={item.label}
                          sx={{
                            ...navItemStyles(selected),
                            justifyContent: 'center',
                            pl: 1.5,
                          }}
                        >
                          <ListItemIcon sx={{ minWidth: 'auto' }}>{navIcons[item.key as keyof typeof navIcons]}</ListItemIcon>
                        </ListItemButton>
                      </Tooltip>
                    )
                  })
                : null
            ) : (
              <Collapse in={sukumadExpanded} unmountOnExit>
                {navigation.sukumad.children.map((item) => {
                  const selected = pathname.startsWith(item.path)
                  return (
                    <ListItemButton
                      key={item.key}
                      selected={selected}
                      onClick={() => handleNavItemClick(item.path)}
                      aria-label={item.label}
                      sx={{
                        ...navItemStyles(selected),
                        justifyContent: 'flex-start',
                        pl: 3.25,
                      }}
                    >
                      <ListItemIcon sx={{ minWidth: 36 }}>{navIcons[item.key as keyof typeof navIcons]}</ListItemIcon>
                      <ListItemText
                        primary={item.label}
                        primaryTypographyProps={{
                          noWrap: true,
                          fontWeight: selected ? 600 : 500,
                        }}
                      />
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
                justifyContent: collapsed && !isMobile ? 'center' : 'flex-start',
                pl: collapsed && !isMobile ? 1.5 : 2.25,
              }}
            >
              <ListItemIcon sx={{ minWidth: collapsed && !isMobile ? 'auto' : 36 }}>
                <AdminPanelSettingsRoundedIcon fontSize="small" />
              </ListItemIcon>
              <ListItemText
                primary={navigation.administration.label}
                primaryTypographyProps={{
                  noWrap: true,
                  fontWeight: 600,
                }}
                sx={{
                  opacity: collapsed && !isMobile ? 0 : 1,
                  transition: theme.transitions.create('opacity', {
                    duration: theme.transitions.duration.shortest,
                  }),
                }}
              />
              {!collapsed || isMobile ? (adminExpanded ? <ExpandLessRoundedIcon fontSize="small" /> : <ExpandMoreRoundedIcon fontSize="small" />) : null}
            </ListItemButton>
            {collapsed && !isMobile ? (
              adminExpanded
                ? navigation.administration.children.map((item) => {
                    const selected = pathname.startsWith(item.path)
                    return (
                      <Tooltip key={item.key} title={item.label} placement="right">
                        <ListItemButton
                          selected={selected}
                          onClick={() => handleNavItemClick(item.path)}
                          aria-label={item.label}
                          sx={{
                            ...navItemStyles(selected),
                            justifyContent: 'center',
                            pl: 1.5,
                          }}
                        >
                          <ListItemIcon sx={{ minWidth: 'auto' }}>{navIcons[item.key as keyof typeof navIcons]}</ListItemIcon>
                        </ListItemButton>
                      </Tooltip>
                    )
                  })
                : null
            ) : (
              <Collapse in={adminExpanded} unmountOnExit>
                {navigation.administration.children.map((item) => {
                  const selected = pathname.startsWith(item.path)
                  return (
                    <ListItemButton
                      key={item.key}
                      selected={selected}
                      onClick={() => handleNavItemClick(item.path)}
                      aria-label={item.label}
                      sx={{
                        ...navItemStyles(selected),
                        justifyContent: 'flex-start',
                        pl: 3.25,
                      }}
                    >
                      <ListItemIcon sx={{ minWidth: 36 }}>{navIcons[item.key as keyof typeof navIcons]}</ListItemIcon>
                      <ListItemText
                        primary={item.label}
                        primaryTypographyProps={{
                          noWrap: true,
                          fontWeight: selected ? 600 : 500,
                        }}
                      />
                    </ListItemButton>
                  )
                })}
              </Collapse>
            )}
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
            {getRouteLabel(pathname)}
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
            {canAccessSettings ? (
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
                {displayName} v0.1.0
              </Typography>
            </Box>
          ) : null}
        </Box>
      </Box>
      <PalettePresetPicker open={appearanceOpen} onClose={() => setAppearanceOpen(false)} />
    </Box>
  )
}
