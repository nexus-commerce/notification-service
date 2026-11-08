package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"strings"
)

//go:embed templates/**/*
var templateFS embed.FS

const pageSuffix = ".page.gohtml"

// Renderer renders HTML pages from embedded templates.
type Renderer struct {
	pages map[string]*template.Template
}

// New parses every embedded page together with the base layout.
func New() (*Renderer, error) {
	files, err := fs.Glob(templateFS, "templates/pages/*"+pageSuffix)
	if err != nil {
		return nil, fmt.Errorf("glob pages: %w", err)
	}
	pages := make(map[string]*template.Template, len(files))
	for _, file := range files {
		base := path.Base(file)
		ts, err := template.New(base).ParseFS(templateFS, "templates/layouts/*.layout.gohtml", file)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", base, err)
		}
		pages[strings.TrimSuffix(base, pageSuffix)] = ts
	}
	return &Renderer{pages: pages}, nil
}

// Render executes the named page with data and returns the HTML.
func (r *Renderer) Render(page string, data any) (string, error) {
	ts, ok := r.pages[page]
	if !ok {
		return "", fmt.Errorf("unknown page %q", page)
	}
	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute page %q: %w", page, err)
	}
	return buf.String(), nil
}
