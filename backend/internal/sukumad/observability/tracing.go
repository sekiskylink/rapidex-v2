package observability

import "basepro/backend/internal/sukumad/traceevent"

func sanitizeEventData(input map[string]any) map[string]any {
	return traceevent.SanitizeData(input)
}

func previewEventData(input map[string]any) string {
	return traceevent.PreviewData(input)
}

func normalizeLevel(value string) string {
	return traceevent.NormalizeLevel(value)
}

func normalizeActorType(value string) string {
	return traceevent.NormalizeActorType(value)
}
