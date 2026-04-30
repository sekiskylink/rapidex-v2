import React from 'react'
import {
  Box,
  Dialog,
  DialogContent,
  DialogTitle,
  InputAdornment,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  TextField,
  Typography,
} from '@mui/material'
import type { NavigationSearchEntry } from '../navigation'
import { SearchRoundedIcon } from '../ui/icons'

interface AppLauncherDialogProps {
  entries: readonly NavigationSearchEntry[]
  open: boolean
  query: string
  onClose: () => void
  onQueryChange: (value: string) => void
  onNavigate: (path: string) => void
  renderIcon: (entry: NavigationSearchEntry) => React.ReactNode
}

export function AppLauncherDialog({
  entries,
  open,
  query,
  onClose,
  onQueryChange,
  onNavigate,
  renderIcon,
}: AppLauncherDialogProps) {
  const [selectedIndex, setSelectedIndex] = React.useState(0)

  React.useEffect(() => {
    if (!open) {
      setSelectedIndex(0)
    }
  }, [open])

  React.useEffect(() => {
    setSelectedIndex((current) => {
      if (entries.length === 0) {
        return 0
      }
      return Math.min(current, entries.length - 1)
    })
  }, [entries])

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      setSelectedIndex((current) => (entries.length === 0 ? 0 : (current + 1) % entries.length))
      return
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      setSelectedIndex((current) =>
        entries.length === 0 ? 0 : (current - 1 + entries.length) % entries.length,
      )
      return
    }
    if (event.key === 'Enter') {
      const selectedEntry = entries[selectedIndex]
      if (!selectedEntry) {
        return
      }
      event.preventDefault()
      onNavigate(selectedEntry.path)
    }
  }

  return (
    <Dialog
      fullWidth
      maxWidth="sm"
      open={open}
      onClose={onClose}
      aria-labelledby="app-launcher-title"
      PaperProps={{ sx: { borderRadius: 3 } }}
    >
      <DialogTitle id="app-launcher-title">Jump To</DialogTitle>
      <DialogContent sx={{ pt: 1 }}>
        <TextField
          autoFocus
          fullWidth
          placeholder="Search apps, pages, settings..."
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          onKeyDown={handleKeyDown}
          inputProps={{ 'aria-label': 'Search apps' }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchRoundedIcon fontSize="small" />
              </InputAdornment>
            ),
          }}
        />
        <List sx={{ mt: 2, py: 0 }}>
          {entries.length > 0 ? (
            entries.map((entry, index) => {
              const selected = index === selectedIndex
              return (
                <ListItemButton
                  key={entry.path}
                  selected={selected}
                  onClick={() => onNavigate(entry.path)}
                  sx={{ borderRadius: 2, mb: 0.5, alignItems: 'flex-start' }}
                >
                  <ListItemIcon sx={{ minWidth: 36, mt: 0.25 }}>{renderIcon(entry)}</ListItemIcon>
                  <ListItemText
                    primary={entry.label}
                    secondary={
                      <Box component="span" sx={{ display: 'flex', flexDirection: 'column' }}>
                        {entry.breadcrumb ? (
                          <Typography component="span" variant="caption" color="text.secondary">
                            {entry.breadcrumb}
                          </Typography>
                        ) : null}
                        <Typography component="span" variant="caption" color="text.secondary">
                          {entry.path}
                        </Typography>
                      </Box>
                    }
                    primaryTypographyProps={{ fontWeight: selected ? 600 : 500 }}
                  />
                </ListItemButton>
              )
            })
          ) : (
            <Box sx={{ px: 1, py: 4 }}>
              <Typography variant="body2" color="text.secondary">
                No accessible routes match that search.
              </Typography>
            </Box>
          )}
        </List>
      </DialogContent>
    </Dialog>
  )
}
