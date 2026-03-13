import React from 'react'
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  Stack,
  TextField,
} from '@mui/material'

export interface RequestServerOption {
  id: number
  name: string
  code: string
}

export interface RequestFormState {
  destinationServerId: string
  destinationServerIdsText: string
  dependencyRequestIdsText: string
  sourceSystem: string
  correlationId: string
  batchId: string
  idempotencyKey: string
  urlSuffix: string
  payloadText: string
  metadataText: string
}

export type RequestFormErrors = Partial<Record<keyof RequestFormState | 'payload' | 'metadata', string>>

interface RequestFormProps {
  open: boolean
  title: string
  form: RequestFormState
  errors: RequestFormErrors
  servers: RequestServerOption[]
  submitting: boolean
  loadingServers: boolean
  errorMessage: string
  testId: string
  submitLabel: string
  onClose: () => void
  onSubmit: () => void
  onChange: (patch: Partial<RequestFormState>) => void
}

export function RequestForm({
  open,
  title,
  form,
  errors,
  servers,
  submitting,
  loadingServers,
  errorMessage,
  testId,
  submitLabel,
  onClose,
  onSubmit,
  onChange,
}: RequestFormProps) {
  return (
    <Dialog open={open} onClose={submitting ? undefined : onClose} maxWidth="md" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }} data-testid={testId}>
          {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              select
              label="Destination Server"
              value={form.destinationServerId}
              onChange={(event) => onChange({ destinationServerId: event.target.value })}
              error={Boolean(errors.destinationServerId)}
              helperText={errors.destinationServerId || (loadingServers ? 'Loading servers...' : 'Select a target server.')}
              disabled={loadingServers}
              fullWidth
            >
              {servers.map((server) => (
                <MenuItem key={server.id} value={String(server.id)}>
                  {server.name} ({server.code})
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="Additional Destination Server IDs"
              value={form.destinationServerIdsText}
              onChange={(event) => onChange({ destinationServerIdsText: event.target.value })}
              error={Boolean(errors.destinationServerIdsText)}
              helperText={errors.destinationServerIdsText || 'Optional comma-separated server IDs for fan-out.'}
              fullWidth
            />
          </Stack>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              label="Dependency Request IDs"
              value={form.dependencyRequestIdsText}
              onChange={(event) => onChange({ dependencyRequestIdsText: event.target.value })}
              error={Boolean(errors.dependencyRequestIdsText)}
              helperText={errors.dependencyRequestIdsText || 'Optional comma-separated request IDs that must complete first.'}
              fullWidth
            />
            <TextField
              label="Source System"
              value={form.sourceSystem}
              onChange={(event) => onChange({ sourceSystem: event.target.value })}
              error={Boolean(errors.sourceSystem)}
              helperText={errors.sourceSystem || 'Optional upstream source identifier.'}
              fullWidth
            />
          </Stack>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              label="Correlation ID"
              value={form.correlationId}
              onChange={(event) => onChange({ correlationId: event.target.value })}
              error={Boolean(errors.correlationId)}
              helperText={errors.correlationId}
              fullWidth
            />
            <TextField
              label="Batch ID"
              value={form.batchId}
              onChange={(event) => onChange({ batchId: event.target.value })}
              error={Boolean(errors.batchId)}
              helperText={errors.batchId}
              fullWidth
            />
          </Stack>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              label="Idempotency Key"
              value={form.idempotencyKey}
              onChange={(event) => onChange({ idempotencyKey: event.target.value })}
              error={Boolean(errors.idempotencyKey)}
              helperText={errors.idempotencyKey}
              fullWidth
            />
            <TextField
              label="URL Suffix"
              value={form.urlSuffix}
              onChange={(event) => onChange({ urlSuffix: event.target.value })}
              error={Boolean(errors.urlSuffix)}
              helperText={errors.urlSuffix || 'Optional path appended downstream during delivery.'}
              fullWidth
            />
          </Stack>
          <TextField
            label="Payload JSON"
            value={form.payloadText}
            onChange={(event) => onChange({ payloadText: event.target.value })}
            error={Boolean(errors.payload)}
            helperText={errors.payload || 'JSON payload submitted into the request lifecycle.'}
            fullWidth
            minRows={10}
            multiline
          />
          <TextField
            label="Metadata JSON"
            value={form.metadataText}
            onChange={(event) => onChange({ metadataText: event.target.value })}
            error={Boolean(errors.metadata)}
            helperText={errors.metadata || 'Optional JSON object of request metadata.'}
            fullWidth
            minRows={6}
            multiline
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={submitting}>
          Cancel
        </Button>
        <Button onClick={onSubmit} disabled={submitting || loadingServers} variant="contained">
          {submitLabel}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
