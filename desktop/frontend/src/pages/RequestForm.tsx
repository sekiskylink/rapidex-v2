import React from 'react'
import {
  Alert,
  Autocomplete,
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
  destinationServerIds: string[]
  dependencyRequestIdsText: string
  sourceSystem: string
  correlationId: string
  batchId: string
  idempotencyKey: string
  payloadFormat: string
  submissionBinding: string
  responseBodyPersistence: string
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
  const payloadLabel = form.payloadFormat === 'text' ? 'Payload Text' : 'Payload JSON'
  const selectedDestination = servers.find((server) => String(server.id) === form.destinationServerId) ?? null
  const selectedAdditionalDestinations = servers.filter((server) => form.destinationServerIds.includes(String(server.id)))
  const payloadHelper =
    errors.payload ||
    (form.payloadFormat === 'text'
      ? form.submissionBinding === 'query'
        ? 'Text payload interpreted as a query string, for example key=value&flag=true.'
        : 'Plain text request body.'
      : form.submissionBinding === 'query'
        ? 'JSON object payload converted into query params during delivery.'
        : 'JSON payload submitted into the request lifecycle.')

  return (
    <Dialog open={open} onClose={submitting ? undefined : onClose} maxWidth="md" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }} data-testid={testId}>
          {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <Autocomplete
              options={servers}
              value={selectedDestination}
              getOptionLabel={(server) => `${server.name} (${server.code})`}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              onChange={(_event, value) =>
                onChange({
                  destinationServerId: value ? String(value.id) : '',
                  destinationServerIds: form.destinationServerIds.filter((id) => id !== (value ? String(value.id) : '')),
                })
              }
              disabled={loadingServers}
              loading={loadingServers}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Destination Server"
                  error={Boolean(errors.destinationServerId)}
                  helperText={errors.destinationServerId || (loadingServers ? 'Loading servers...' : 'Select a target server.')}
                />
              )}
              fullWidth
            />
            <Autocomplete
              multiple
              options={servers.filter((server) => String(server.id) !== form.destinationServerId)}
              value={selectedAdditionalDestinations}
              getOptionLabel={(server) => `${server.name} (${server.code})`}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              onChange={(_event, values) => onChange({ destinationServerIds: values.map((server) => String(server.id)) })}
              disabled={loadingServers}
              loading={loadingServers}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Additional Destination Servers"
                  error={Boolean(errors.destinationServerIds)}
                  helperText={errors.destinationServerIds || 'Optional fan-out destination servers.'}
                />
              )}
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
              select
              label="Payload Format"
              value={form.payloadFormat}
              onChange={(event) => onChange({ payloadFormat: event.target.value })}
              error={Boolean(errors.payloadFormat)}
              helperText={errors.payloadFormat || 'How the payload is stored and interpreted.'}
              fullWidth
            >
              <MenuItem value="json">JSON</MenuItem>
              <MenuItem value="text">Text</MenuItem>
            </TextField>
          </Stack>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              select
              label="Send As"
              value={form.submissionBinding}
              onChange={(event) => onChange({ submissionBinding: event.target.value })}
              error={Boolean(errors.submissionBinding)}
              helperText={errors.submissionBinding || 'Choose whether the payload becomes the request body or query params.'}
              fullWidth
            >
              <MenuItem value="body">Request Body</MenuItem>
              <MenuItem value="query">Query Params</MenuItem>
            </TextField>
            <TextField
              select
              label="Response Body"
              value={form.responseBodyPersistence}
              onChange={(event) => onChange({ responseBodyPersistence: event.target.value })}
              error={Boolean(errors.responseBodyPersistence)}
              helperText={errors.responseBodyPersistence || 'Override the destination response body saving policy.'}
              fullWidth
            >
              <MenuItem value="">Server default</MenuItem>
              <MenuItem value="filter">Use response filter</MenuItem>
              <MenuItem value="save">Always save</MenuItem>
              <MenuItem value="discard">Never save</MenuItem>
            </TextField>
          </Stack>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
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
            label={payloadLabel}
            value={form.payloadText}
            onChange={(event) => onChange({ payloadText: event.target.value })}
            error={Boolean(errors.payload)}
            helperText={payloadHelper}
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
