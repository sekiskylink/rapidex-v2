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
  currentPath: string
  entries: readonly NavigationSearchEntry[]
  open: boolean
  onClose: () => void
  onNavigate: (path: string) => void
  renderIcon: (entry: NavigationSearchEntry) => React.ReactNode
}

export function AppLauncherDialog({
  currentPath,
  entries,
  open,
  onClose,
  onNavigate,
  renderIcon,
}: AppLauncherDialogProps) {
  const [query, setQuery] = React.useState('')
  const [selectedIndex, setSelectedIndex] = React.useState(0)

  React.useEffect(() => {
    if (!open) {
      setQuery('')
      setSelectedIndex(0)
    }
  }, [open])

  const filteredEntries = React.useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase()
    const nextEntries = normalizedQuery
      ? entries.filter((entry) => entry.searchText.includes(normalizedQuery))
      : entries

    return [...nextEntries].sort((left, right) => {
      if (left.path === currentPath && right.path !== currentPath) {
        return -1
      }
      if (right.path === currentPath && left.path !== currentPath) {
        return 1
      }
      return left.label.localeCompare(right.label)
    })
  }, [currentPath, entries, query])

  React.useEffect(() => {
    setSelectedIndex((current) => {
      if (filteredEntries.length === 0) {
        return 0
      }
      return Math.min(current, filteredEntries.length - 1)
    })
  }, [filteredEntries])

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      setSelectedIndex((current) => (filteredEntries.length === 0 ? 0 : (current + 1) % filteredEntries.length))
      return
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      setSelectedIndex((current) =>
        filteredEntries.length === 0 ? 0 : (current - 1 + filteredEntries.length) % filteredEntries.length,
      )
      return
    }
    if (event.key === 'Enter') {
      const selectedEntry = filteredEntries[selectedIndex]
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
      <DialogTitle id="app-launcher-title">App Search</DialogTitle>
      <DialogContent sx={{ pt: 1 }}>
        <TextField
          autoFocus
          fullWidth
          label="Search apps"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={handleKeyDown}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchRoundedIcon fontSize="small" />
              </InputAdornment>
            ),
          }}
        />
        <List sx={{ mt: 2, py: 0 }}>
          {filteredEntries.length > 0 ? (
            filteredEntries.map((entry, index) => {
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
