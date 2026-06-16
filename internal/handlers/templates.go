package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
)

var templates *template.Template

func InitTemplates(templateDir string) error {
	var err error
	templates, err = template.ParseGlob(filepath.Join(templateDir, "*.html"))
	if err != nil {
		return err
	}
	return nil
}

func renderTemplate(w http.ResponseWriter, name string, data any) error {
	return templates.ExecuteTemplate(w, name, data)
}
