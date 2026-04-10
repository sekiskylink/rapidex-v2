import React from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Divider,
  List,
  ListItemButton,
  ListItemText,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { apiRequest } from '../lib/api'

interface DocumentationSummary {
  slug: string
  title: string
  sourcePath: string
  updatedAt?: string | null
}

interface DocumentationListResponse {
  items: DocumentationSummary[]
}

interface DocumentationDetail extends DocumentationSummary {
  content: string
}

function formatUpdatedAt(value?: string | null) {
  if (!value) {
    return ''
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  return date.toLocaleString()
}

function MarkdownViewer({ content }: { content: string }) {
  return (
    <Box
      sx={{
        '& h1, & h2, & h3': { scrollMarginTop: 96 },
        '& table': { width: '100%', borderCollapse: 'collapse', my: 2 },
        '& th, & td': { border: 1, borderColor: 'divider', px: 1.25, py: 1, verticalAlign: 'top' },
        '& th': { bgcolor: 'action.hover', fontWeight: 700 },
        '& pre': {
          overflowX: 'auto',
          bgcolor: 'grey.950',
          color: 'grey.100',
          borderRadius: 1,
          p: 2,
        },
        '& code': {
          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
          fontSize: '0.9em',
        },
        '& :not(pre) > code': {
          bgcolor: 'action.hover',
          color: 'text.primary',
          borderRadius: 0.75,
          px: 0.5,
          py: 0.15,
        },
        '& blockquote': {
          borderLeft: 4,
          borderColor: 'primary.main',
          color: 'text.secondary',
          my: 2,
          mx: 0,
          pl: 2,
        },
        '& img': { maxWidth: '100%' },
      }}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ children }) => (
            <Typography variant="h4" component="h1" sx={{ fontWeight: 700, mb: 2 }}>
              {children}
            </Typography>
          ),
          h2: ({ children }) => (
            <Typography variant="h5" component="h2" sx={{ fontWeight: 700, mt: 4, mb: 1.5 }}>
              {children}
            </Typography>
          ),
          h3: ({ children }) => (
            <Typography variant="h6" component="h3" sx={{ fontWeight: 700, mt: 3, mb: 1 }}>
              {children}
            </Typography>
          ),
          p: ({ children }) => (
            <Typography component="p" sx={{ mb: 1.5, lineHeight: 1.75 }}>
              {children}
            </Typography>
          ),
          a: ({ href, children }) => (
            <Typography component="a" href={href} target={href?.startsWith('http') ? '_blank' : undefined} rel="noreferrer" sx={{ color: 'primary.main' }}>
              {children}
            </Typography>
          ),
          li: ({ children }) => (
            <Typography component="li" sx={{ mb: 0.75 }}>
              {children}
            </Typography>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </Box>
  )
}

export function DocumentationPage() {
  const [items, setItems] = React.useState<DocumentationSummary[]>([])
  const [selectedSlug, setSelectedSlug] = React.useState('')
  const [document, setDocument] = React.useState<DocumentationDetail | null>(null)
  const [filter, setFilter] = React.useState('')
  const [loadingList, setLoadingList] = React.useState(true)
  const [loadingDocument, setLoadingDocument] = React.useState(false)
  const [errorMessage, setErrorMessage] = React.useState('')

  React.useEffect(() => {
    let active = true
    setLoadingList(true)
    setErrorMessage('')
    apiRequest<DocumentationListResponse>('/documentation')
      .then((payload) => {
        if (!active) {
          return
        }
        const nextItems = payload.items ?? []
        setItems(nextItems)
        setSelectedSlug((current) => current || nextItems[0]?.slug || '')
      })
      .catch(() => {
        if (active) {
          setErrorMessage('Unable to load documentation.')
        }
      })
      .finally(() => {
        if (active) {
          setLoadingList(false)
        }
      })
    return () => {
      active = false
    }
  }, [])

  React.useEffect(() => {
    if (!selectedSlug) {
      setDocument(null)
      return
    }
    let active = true
    setLoadingDocument(true)
    setErrorMessage('')
    apiRequest<DocumentationDetail>(`/documentation/${encodeURIComponent(selectedSlug)}`)
      .then((payload) => {
        if (active) {
          setDocument(payload)
        }
      })
      .catch(() => {
        if (active) {
          setErrorMessage('Unable to load the selected document.')
          setDocument(null)
        }
      })
      .finally(() => {
        if (active) {
          setLoadingDocument(false)
        }
      })
    return () => {
      active = false
    }
  }, [selectedSlug])

  const filteredItems = React.useMemo(() => {
    const query = filter.trim().toLowerCase()
    if (!query) {
      return items
    }
    return items.filter((item) => item.title.toLowerCase().includes(query) || item.sourcePath.toLowerCase().includes(query))
  }, [filter, items])

  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h4" component="h1" sx={{ fontWeight: 700 }}>
          Documentation
        </Typography>
        <Typography color="text.secondary">Operational notes and implementation references.</Typography>
      </Box>

      {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}

      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', md: '320px minmax(0, 1fr)' },
          gap: 3,
          alignItems: 'start',
        }}
      >
        <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
          <Box sx={{ p: 2 }}>
            <TextField fullWidth label="Search documents" value={filter} onChange={(event) => setFilter(event.target.value)} size="small" />
          </Box>
          <Divider />
          {loadingList ? (
            <Stack alignItems="center" sx={{ py: 4 }}>
              <CircularProgress size={28} />
            </Stack>
          ) : (
            <List disablePadding sx={{ maxHeight: { md: 'calc(100vh - 260px)' }, overflow: 'auto' }}>
              {filteredItems.map((item) => (
                <ListItemButton key={item.slug} selected={item.slug === selectedSlug} onClick={() => setSelectedSlug(item.slug)}>
                  <ListItemText primary={item.title} secondary={item.sourcePath} primaryTypographyProps={{ fontWeight: item.slug === selectedSlug ? 700 : 500 }} />
                </ListItemButton>
              ))}
              {filteredItems.length === 0 ? (
                <Box sx={{ p: 2 }}>
                  <Typography color="text.secondary">No documents found.</Typography>
                </Box>
              ) : null}
            </List>
          )}
        </Box>

        <Box sx={{ minWidth: 0, border: 1, borderColor: 'divider', borderRadius: 1, p: { xs: 2, md: 3 } }}>
          {loadingDocument ? (
            <Stack direction="row" alignItems="center" spacing={1.5}>
              <CircularProgress size={24} />
              <Typography>Loading document...</Typography>
            </Stack>
          ) : document ? (
            <Stack spacing={2}>
              <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={1}>
                <Box>
                  <Typography variant="overline" color="text.secondary">
                    {document.sourcePath}
                  </Typography>
                  {formatUpdatedAt(document.updatedAt) ? (
                    <Typography color="text.secondary" variant="body2">
                      Updated {formatUpdatedAt(document.updatedAt)}
                    </Typography>
                  ) : null}
                </Box>
                <Button size="small" variant="outlined" onClick={() => setFilter('')}>
                  Clear Search
                </Button>
              </Stack>
              <Divider />
              <MarkdownViewer content={document.content} />
            </Stack>
          ) : (
            <Typography color="text.secondary">Select a document.</Typography>
          )}
        </Box>
      </Box>
    </Stack>
  )
}

