package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

func parseTemplate(filename string) *appTemplate {
	tmpl := template.Must(template.ParseFiles("templates/base.html"))
	path := filepath.Join("templates", filename)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Errorf("could not read template: %v", err))
	}
	template.Must(tmpl.New("body").Parse(string(b)))
	return &appTemplate{t: tmpl.Lookup("base.html")}
}

type appTemplate struct {
	t *template.Template
}

func (tmpl *appTemplate) Execute(n *Novelshelf, w http.ResponseWriter, r *http.Request, data interface{}) *appError {
	d := struct {
		Data interface{}
	}{
		Data: data,
	}

	if err := tmpl.t.Execute(w, d); err != nil {
		return n.appErrorf(r, err, "could not write template: %v", err)
	}
	return nil
}
