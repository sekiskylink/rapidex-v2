package docbrowser

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"basepro/backend/internal/apperror"
)

func writeDoc(t *testing.T, root string, name string, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write doc: %v", err)
	}
}

func TestServiceListsConfiguredDocumentsInOrder(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "b.md", "# B")
	writeDoc(t, root, "a.md", "# A")
	service := NewService(func() SourceConfig {
		return SourceConfig{
			RootPath: root,
			Files: []SourceFile{
				{Slug: "b", Title: "B", Path: "b.md", Order: 20},
				{Slug: "a", Title: "A", Path: "a.md", Order: 10},
			},
		}
	})

	items, err := service.ListDocuments(context.Background())
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(items) != 2 || items[0].Slug != "a" || items[1].Slug != "b" {
		t.Fatalf("unexpected document order: %+v", items)
	}
	if items[0].SourcePath != "a.md" || items[0].UpdatedAt == nil {
		t.Fatalf("unexpected summary: %+v", items[0])
	}
}

func TestServiceGetsConfiguredDocument(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "overview.md", "# Overview\n\nBody")
	service := NewService(func() SourceConfig {
		return SourceConfig{
			RootPath: root,
			Files: []SourceFile{
				{Slug: "overview", Title: "Overview", Path: "overview.md"},
			},
		}
	})

	item, err := service.GetDocument(context.Background(), "overview")
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if item.Title != "Overview" || item.Content != "# Overview\n\nBody" {
		t.Fatalf("unexpected document: %+v", item)
	}
}

func TestServiceRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	service := NewService(func() SourceConfig {
		return SourceConfig{
			RootPath: root,
			Files: []SourceFile{
				{Slug: "escape", Title: "Escape", Path: "../secrets.md"},
			},
		}
	})

	_, err := service.ListDocuments(context.Background())
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeValidationFailed {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestServiceRejectsNonMarkdownPaths(t *testing.T) {
	root := t.TempDir()
	service := NewService(func() SourceConfig {
		return SourceConfig{
			RootPath: root,
			Files: []SourceFile{
				{Slug: "text", Title: "Text", Path: "text.txt"},
			},
		}
	})

	_, err := service.ListDocuments(context.Background())
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeValidationFailed {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestServiceReturnsNotFoundForUnknownSlug(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "overview.md", "# Overview")
	service := NewService(func() SourceConfig {
		return SourceConfig{
			RootPath: root,
			Files: []SourceFile{
				{Slug: "overview", Title: "Overview", Path: "overview.md"},
			},
		}
	})

	_, err := service.GetDocument(context.Background(), "missing")
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "DOCUMENT_NOT_FOUND" {
		t.Fatalf("expected not found error, got %v", err)
	}
}
