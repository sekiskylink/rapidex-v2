import React from 'react'
import { Box, Paper, Stack, Typography, useTheme } from '@mui/material'

export interface AuthBranding {
  appDisplayName: string
  loginImageUrl: string | null
}

interface AuthSplitLayoutProps {
  branding: AuthBranding
  panelTitle: string
  panelSubtitle: string
  children: React.ReactNode
}

export function AuthSplitLayout({ branding, panelTitle, panelSubtitle, children }: AuthSplitLayoutProps) {
  const theme = useTheme()
  const [showImage, setShowImage] = React.useState(Boolean(branding.loginImageUrl))

  React.useEffect(() => {
    setShowImage(Boolean(branding.loginImageUrl))
  }, [branding.loginImageUrl])

  return (
    <Box
      sx={{
        minHeight: '100vh',
        px: { xs: 2, sm: 3, md: 4 },
        py: { xs: 3, md: 5 },
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background:
          theme.palette.mode === 'light'
            ? 'radial-gradient(circle at 20% 15%, rgba(59, 130, 246, 0.15), transparent 45%), radial-gradient(circle at 80% 85%, rgba(14, 165, 233, 0.12), transparent 40%)'
            : 'radial-gradient(circle at 20% 15%, rgba(56, 189, 248, 0.25), transparent 45%), radial-gradient(circle at 80% 85%, rgba(14, 165, 233, 0.2), transparent 40%)',
      }}
    >
      <Paper
        elevation={0}
        sx={{
          width: '100%',
          maxWidth: 1080,
          borderRadius: 4,
          overflow: 'hidden',
          border: '1px solid',
          borderColor: 'divider',
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
          backdropFilter: 'blur(6px)',
        }}
      >
        <Box
          sx={{
            p: { xs: 3, md: 5 },
            background:
              theme.palette.mode === 'light'
                ? 'linear-gradient(145deg, rgba(2, 132, 199, 0.08), rgba(14, 165, 233, 0.2))'
                : 'linear-gradient(145deg, rgba(8, 47, 73, 0.65), rgba(12, 74, 110, 0.5))',
            borderRight: { xs: 0, md: '1px solid' },
            borderBottom: { xs: '1px solid', md: 0 },
            borderColor: 'divider',
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <Stack spacing={3} sx={{ width: '100%' }}>
            <Typography variant="overline" color="text.secondary">
              Platform Access
            </Typography>
            <Typography variant="h3" component="h1" sx={{ fontSize: { xs: '1.75rem', md: '2.2rem' } }}>
              {branding.appDisplayName}
            </Typography>
            {showImage && branding.loginImageUrl ? (
              <Box
                component="img"
                src={branding.loginImageUrl}
                alt={`${branding.appDisplayName} login illustration`}
                onError={() => setShowImage(false)}
                sx={{
                  width: '100%',
                  maxHeight: { xs: 220, md: 320 },
                  objectFit: 'contain',
                  borderRadius: 3,
                  border: '1px solid',
                  borderColor: 'divider',
                  backgroundColor: 'background.paper',
                  p: 2,
                }}
              />
            ) : (
              <Box
                sx={{
                  display: 'grid',
                  placeItems: 'center',
                  minHeight: { xs: 180, md: 260 },
                  borderRadius: 3,
                  border: '1px dashed',
                  borderColor: 'divider',
                  backgroundColor: 'background.paper',
                  px: 3,
                }}
              >
                <Stack spacing={1.25} alignItems="center">
                  <Box
                    sx={{
                      width: 64,
                      height: 64,
                      borderRadius: '50%',
                      display: 'grid',
                      placeItems: 'center',
                      bgcolor: 'primary.main',
                      color: 'primary.contrastText',
                      fontWeight: 700,
                      fontSize: 24,
                    }}
                  >
                    {branding.appDisplayName.slice(0, 1).toUpperCase()}
                  </Box>
                  <Typography variant="subtitle1">{branding.appDisplayName}</Typography>
                </Stack>
              </Box>
            )}
          </Stack>
        </Box>

        <Box sx={{ p: { xs: 3, md: 5 }, display: 'flex', alignItems: 'center' }}>
          <Stack spacing={3} sx={{ width: '100%' }}>
            <Box>
              <Typography variant="h5" component="h2" gutterBottom>
                {panelTitle}
              </Typography>
              <Typography color="text.secondary">{panelSubtitle}</Typography>
            </Box>
            {children}
          </Stack>
        </Box>
      </Paper>
    </Box>
  )
}
