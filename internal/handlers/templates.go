package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
)

var templates = map[string]*template.Template{}

func InitTemplates(templateDir string) error {
	pages := []string{"login.html", "dashboard.html", "create.html", "404.html"}
	for _, page := range pages {
		t, err := template.ParseFiles(
			filepath.Join(templateDir, "base.html"),
			filepath.Join(templateDir, page),
		)
		if err != nil {
			return err
		}
		templates[page] = t
	}
	return nil
}

func renderTemplate(w http.ResponseWriter, page string, data any) error {
	return templates[page].ExecuteTemplate(w, "base.html", data)
}
