package docbrowser

import "time"

type SourceFile struct {
	Slug  string `mapstructure:"slug"`
	Title string `mapstructure:"title"`
	Path  string `mapstructure:"path"`
	Order int    `mapstructure:"order"`
}

type SourceConfig struct {
	RootPath string
	Files    []SourceFile
}

type DocumentSummary struct {
	Slug       string     `json:"slug"`
	Title      string     `json:"title"`
	SourcePath string     `json:"sourcePath"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
}

type DocumentDetail struct {
	DocumentSummary
	Content string `json:"content"`
}
