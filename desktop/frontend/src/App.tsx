import React from 'react'
import { Alert, Snackbar } from '@mui/material'
import { Outlet, useNavigate } from '@tanstack/react-router'
import { onSessionExpired } from './auth/session'
import { subscribeNotifications, type Notification } from './notifications/store'

function App() {
  const navigate = useNavigate()
  const [notification, setNotification] = React.useState<Notification | null>(null)

  React.useEffect(() => {
    return onSessionExpired((reason) => {
      if (reason === 'expired') {
        setNotification({ message: 'Session expired. Please log in again.', severity: 'warning' })
        void navigate({ to: '/login', replace: true })
        return
      }

      setNotification({ message: 'Unable to reach API. Check your connection.', severity: 'error' })
    })
  }, [navigate])

  React.useEffect(() => subscribeNotifications(setNotification), [])

  return (
    <>
      <Outlet />
      <Snackbar
        open={Boolean(notification)}
        autoHideDuration={4000}
        onClose={() => setNotification(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity={notification?.severity ?? 'info'} onClose={() => setNotification(null)}>
          {notification?.message}
        </Alert>
      </Snackbar>
    </>
  )
}

export default App
