package templates

import (
	"embed"
	"html/template"
	"sync"
)

//go:embed *.gohtml
var files embed.FS

var (
	once    sync.Once
	tpl     *template.Template
	initErr error
)

// Shared returns the parsed template set.
func Shared() (*template.Template, error) {
	once.Do(func() {
		tpl, initErr = template.New("root").Funcs(template.FuncMap{}).ParseFS(files, "*.gohtml")
	})
	return tpl, initErr
}
