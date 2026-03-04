import React from 'react'
import { Outlet } from '@tanstack/react-router'
import { AuthProvider } from './auth/AuthProvider'
import { AppThemeProvider } from './ui/theme/AppThemeProvider'

export default function App() {
  return (
    <AppThemeProvider>
      <AuthProvider>
        <Outlet />
      </AuthProvider>
    </AppThemeProvider>
  )
}
