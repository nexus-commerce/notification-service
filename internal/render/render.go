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

type Renderer struct {
	pages map[string]*template.Template
}

// New parses the embedded pages. contactAddress is the support address
// shown in the footer of every email.
func New(contactAddress string) (*Renderer, error) {
	files, err := fs.Glob(templateFS, "templates/pages/*"+pageSuffix)
	if err != nil {
		return nil, fmt.Errorf("glob pages: %w", err)
	}
	funcs := template.FuncMap{"contactAddress": func() string { return contactAddress }}
	pages := make(map[string]*template.Template, len(files))
	for _, file := range files {
		base := path.Base(file)
		ts, err := template.New(base).Funcs(funcs).ParseFS(templateFS, "templates/layouts/*.layout.gohtml", file)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", base, err)
		}
		pages[strings.TrimSuffix(base, pageSuffix)] = ts
	}
	return &Renderer{pages: pages}, nil
}

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
