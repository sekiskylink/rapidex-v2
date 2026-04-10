package docbrowser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"basepro/backend/internal/apperror"
)

type ConfigProvider func() SourceConfig

type Service struct {
	configProvider ConfigProvider
}

func NewService(configProvider ConfigProvider) *Service {
	return &Service{configProvider: configProvider}
}

func (s *Service) ListDocuments(ctx context.Context) ([]DocumentSummary, error) {
	cfg := s.config()
	entries, err := resolveEntries(cfg)
	if err != nil {
		return nil, err
	}

	items := make([]DocumentSummary, 0, len(entries))
	for _, entry := range entries {
		summary, err := entry.summary(ctx)
		if err != nil {
			return nil, err
		}
		items = append(items, summary)
	}
	return items, nil
}

func (s *Service) GetDocument(ctx context.Context, slug string) (DocumentDetail, error) {
	trimmedSlug := strings.TrimSpace(slug)
	if trimmedSlug == "" {
		return DocumentDetail{}, validation("slug", "is required")
	}

	cfg := s.config()
	entries, err := resolveEntries(cfg)
	if err != nil {
		return DocumentDetail{}, err
	}

	for _, entry := range entries {
		if entry.file.Slug != trimmedSlug {
			continue
		}
		if err := ctx.Err(); err != nil {
			return DocumentDetail{}, err
		}
		content, err := os.ReadFile(entry.fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				return DocumentDetail{}, notFound()
			}
			return DocumentDetail{}, fmt.Errorf("read documentation file: %w", err)
		}
		summary, err := entry.summary(ctx)
		if err != nil {
			return DocumentDetail{}, err
		}
		return DocumentDetail{
			DocumentSummary: summary,
			Content:         string(content),
		}, nil
	}

	return DocumentDetail{}, notFound()
}

func (s *Service) config() SourceConfig {
	if s == nil || s.configProvider == nil {
		return SourceConfig{}
	}
	return s.configProvider()
}

type resolvedEntry struct {
	rootFullPath string
	fullPath     string
	file         SourceFile
}

func resolveEntries(cfg SourceConfig) ([]resolvedEntry, error) {
	root := strings.TrimSpace(cfg.RootPath)
	if root == "" {
		return nil, validation("rootPath", "is required")
	}
	rootFullPath, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return nil, fmt.Errorf("resolve documentation root: %w", err)
	}

	entries := make([]resolvedEntry, 0, len(cfg.Files))
	slugs := map[string]struct{}{}
	for _, file := range cfg.Files {
		slug := strings.TrimSpace(file.Slug)
		if slug == "" {
			return nil, validation("slug", "is required")
		}
		if _, ok := slugs[slug]; ok {
			return nil, validation("slug", "must be unique")
		}
		slugs[slug] = struct{}{}

		title := strings.TrimSpace(file.Title)
		if title == "" {
			return nil, validation("title", "is required")
		}

		sourcePath := filepath.Clean(strings.TrimSpace(file.Path))
		if sourcePath == "" || sourcePath == "." {
			return nil, validation("path", "is required")
		}
		if filepath.IsAbs(sourcePath) {
			return nil, validation("path", "must be relative to the documentation root")
		}
		if strings.ToLower(filepath.Ext(sourcePath)) != ".md" {
			return nil, validation("path", "must reference a markdown file")
		}

		fullPath := filepath.Clean(filepath.Join(rootFullPath, sourcePath))
		if !isWithinRoot(rootFullPath, fullPath) {
			return nil, validation("path", "must stay under the documentation root")
		}
		file.Slug = slug
		file.Title = title
		file.Path = sourcePath
		entries = append(entries, resolvedEntry{
			rootFullPath: rootFullPath,
			fullPath:     fullPath,
			file:         file,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].file.Order != entries[j].file.Order {
			return entries[i].file.Order < entries[j].file.Order
		}
		return strings.ToLower(entries[i].file.Title) < strings.ToLower(entries[j].file.Title)
	})
	return entries, nil
}

func (e resolvedEntry) summary(ctx context.Context) (DocumentSummary, error) {
	if err := ctx.Err(); err != nil {
		return DocumentSummary{}, err
	}
	info, err := os.Stat(e.fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DocumentSummary{}, notFound()
		}
		return DocumentSummary{}, fmt.Errorf("stat documentation file: %w", err)
	}
	if info.IsDir() {
		return DocumentSummary{}, validation("path", "must reference a file")
	}
	updatedAt := info.ModTime().UTC()
	return DocumentSummary{
		Slug:       e.file.Slug,
		Title:      e.file.Title,
		SourcePath: e.file.Path,
		UpdatedAt:  &updatedAt,
	}, nil
}

func isWithinRoot(root string, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (!strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != "..")
}

func validation(field string, message string) error {
	return apperror.ValidationWithDetails("documentation config invalid", map[string]any{field: []string{message}})
}

func notFound() error {
	return &apperror.AppError{
		HTTPStatus: 404,
		Code:       "DOCUMENT_NOT_FOUND",
		Message:    "Documentation document not found",
		Details:    map[string]any{},
	}
}
