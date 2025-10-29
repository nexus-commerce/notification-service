package render

import (
	"embed"
	"html/template"
	"io/fs"
	"path/filepath"
)

//go:embed templates/**/*
var templateFS embed.FS

func NewTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := fs.Glob(templateFS, "templates/pages/*.page.gohtml")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		ts, err := template.New(name).ParseFS(templateFS,
			"templates/layouts/*.layout.gohtml", page)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
