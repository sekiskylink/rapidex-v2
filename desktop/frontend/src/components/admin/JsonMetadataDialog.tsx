import React from 'react'
import { Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Stack, Typography } from '@mui/material'
import ContentCopyRoundedIcon from '@mui/icons-material/ContentCopyRounded'

interface JsonMetadataDialogProps {
  open: boolean
  title?: string
  metadata: unknown
  emptyMessage?: string
  invalidMessage?: string
  onClose: () => void
  onCopied?: () => void
}

function asPrettyJson(metadata: unknown) {
  if (metadata == null || metadata === '') {
    return { kind: 'empty' as const, value: '' }
  }

  if (typeof metadata === 'string') {
    try {
      const parsed = JSON.parse(metadata)
      return { kind: 'json' as const, value: JSON.stringify(parsed, null, 2) }
    } catch {
      return { kind: 'invalid' as const, value: metadata }
    }
  }

  try {
    return { kind: 'json' as const, value: JSON.stringify(metadata, null, 2) }
  } catch {
    return { kind: 'invalid' as const, value: String(metadata) }
  }
}

export function JsonMetadataDialog({
  open,
  title = 'Audit Metadata',
  metadata,
  emptyMessage = 'No metadata available.',
  invalidMessage = 'Metadata is not valid JSON.',
  onClose,
  onCopied,
}: JsonMetadataDialogProps) {
  const parsed = React.useMemo(() => asPrettyJson(metadata), [metadata])

  const onCopy = async () => {
    const valueToCopy = parsed.kind === 'empty' ? '' : parsed.value
    if (typeof navigator === 'undefined' || !navigator.clipboard) {
      return
    }
    await navigator.clipboard.writeText(valueToCopy)
    onCopied?.()
  }

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <Stack spacing={1.5}>
          {parsed.kind === 'empty' ? <Typography color="text.secondary">{emptyMessage}</Typography> : null}
          {parsed.kind === 'invalid' ? <Typography color="warning.main">{invalidMessage}</Typography> : null}
          {parsed.kind !== 'empty' ? (
            <Box
              sx={{
                position: 'relative',
                maxHeight: 420,
                overflow: 'auto',
                p: 2,
                borderRadius: 2,
                border: '1px solid',
                borderColor: 'divider',
                bgcolor: 'background.default',
                backgroundImage: (theme) =>
                  `linear-gradient(180deg, ${theme.palette.action.hover} 0%, ${theme.palette.background.default} 100%)`,
                boxShadow: (theme) => `inset 0 0 0 1px ${theme.palette.action.selected}`,
                '&::before': {
                  content: '""',
                  position: 'absolute',
                  left: 0,
                  top: 10,
                  bottom: 10,
                  width: 4,
                  borderRadius: '0 999px 999px 0',
                  bgcolor: 'primary.main',
                },
              }}
            >
              <Stack spacing={1}>
                <Chip
                  label={parsed.kind === 'invalid' ? 'Raw body' : 'Formatted JSON'}
                  size="small"
                  color={parsed.kind === 'invalid' ? 'warning' : 'primary'}
                  sx={{ alignSelf: 'flex-start' }}
                />
                <Typography
                  component="pre"
                  sx={{
                    m: 0,
                    pl: 1,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
                    fontSize: 13,
                    lineHeight: 1.7,
                  }}
                >
                  {parsed.value}
                </Typography>
              </Stack>
            </Box>
          ) : null}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button startIcon={<ContentCopyRoundedIcon />} onClick={() => void onCopy()} disabled={parsed.kind === 'empty'}>
          Copy
        </Button>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}
