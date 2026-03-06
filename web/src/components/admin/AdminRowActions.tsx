import React from 'react'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
} from '@mui/material'
import { DeleteOutlineRoundedIcon, EditRoundedIcon, MoreVertRoundedIcon, VisibilityRoundedIcon } from '../../ui/icons'

export interface AdminRowAction {
  id: string
  label: string
  onClick: () => void
  visible?: boolean
  disabled?: boolean
  destructive?: boolean
  confirmTitle?: string
  confirmMessage?: string
  icon?: 'view' | 'edit' | 'delete' | React.ReactNode
}

interface AdminRowActionsProps {
  actions: AdminRowAction[]
  rowLabel?: string
}

function resolveIcon(icon: AdminRowAction['icon']) {
  if (icon === 'view') {
    return <VisibilityRoundedIcon fontSize="small" />
  }
  if (icon === 'edit') {
    return <EditRoundedIcon fontSize="small" />
  }
  if (icon === 'delete') {
    return <DeleteOutlineRoundedIcon fontSize="small" />
  }
  return icon ?? null
}

export function AdminRowActions({ actions, rowLabel }: AdminRowActionsProps) {
  const visibleActions = actions.filter((action) => action.visible !== false)
  const [anchorEl, setAnchorEl] = React.useState<HTMLElement | null>(null)
  const [pendingAction, setPendingAction] = React.useState<AdminRowAction | null>(null)

  if (visibleActions.length === 0) {
    return null
  }

  const closeMenu = () => {
    setAnchorEl(null)
  }

  const executeAction = (action: AdminRowAction) => {
    closeMenu()
    action.onClick()
  }

  const openConfirmation = (action: AdminRowAction) => {
    closeMenu()
    setPendingAction(action)
  }

  return (
    <>
      <IconButton
        size="small"
        aria-label={rowLabel ? `Actions for ${rowLabel}` : 'Row actions'}
        onClick={(event) => setAnchorEl(event.currentTarget)}
      >
        <MoreVertRoundedIcon fontSize="small" />
      </IconButton>
      <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={closeMenu}>
        {visibleActions.map((action) => {
          const iconNode = resolveIcon(action.icon)
          const requiresConfirm = action.destructive || Boolean(action.confirmTitle) || Boolean(action.confirmMessage)
          return (
            <MenuItem
              key={action.id}
              disabled={action.disabled}
              onClick={() => {
                if (requiresConfirm) {
                  openConfirmation(action)
                  return
                }
                executeAction(action)
              }}
            >
              {iconNode ? <ListItemIcon>{iconNode}</ListItemIcon> : null}
              <ListItemText>{action.label}</ListItemText>
            </MenuItem>
          )
        })}
      </Menu>

      <Dialog open={Boolean(pendingAction)} onClose={() => setPendingAction(null)} maxWidth="xs" fullWidth>
        <DialogTitle>{pendingAction?.confirmTitle ?? 'Confirm action'}</DialogTitle>
        <DialogContent>
          {pendingAction?.confirmMessage ??
            `Are you sure you want to ${pendingAction?.label.toLowerCase()}${rowLabel ? ` for ${rowLabel}` : ''}?`}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPendingAction(null)}>Cancel</Button>
          <Button
            color={pendingAction?.destructive ? 'error' : 'primary'}
            variant="contained"
            onClick={() => {
              if (!pendingAction) {
                return
              }
              const action = pendingAction
              setPendingAction(null)
              action.onClick()
            }}
          >
            Confirm
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
