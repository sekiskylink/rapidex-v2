import React from 'react'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, Typography } from '@mui/material'
import ContentCopyRoundedIcon from '@mui/icons-material/ContentCopyRounded'

interface JsonMetadataDialogProps {
  open: boolean
  title?: string
  metadata: unknown
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
          {parsed.kind === 'empty' ? <Typography color="text.secondary">No metadata available.</Typography> : null}
          {parsed.kind === 'invalid' ? <Typography color="warning.main">Metadata is not valid JSON.</Typography> : null}
          {parsed.kind !== 'empty' ? (
            <Typography
              component="pre"
              sx={{
                m: 0,
                maxHeight: 420,
                overflow: 'auto',
                p: 1.5,
                borderRadius: 1,
                border: '1px solid',
                borderColor: 'divider',
                bgcolor: 'background.default',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
                fontSize: 13,
              }}
            >
              {parsed.value}
            </Typography>
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
