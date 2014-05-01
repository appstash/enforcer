package main

import (
    "html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var html *template.Template

func init() {
	filenames := []string{}
	err := filepath.Walk("html", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && filepath.Ext(path) == ".gohtml" {
			filenames = append(filenames, path)
		}

		return nil
	})
	if err != nil {
		log.Fatalln(err)
	}

	if len(filenames) == 0 {
		return
	}

	html, err = template.ParseFiles(filenames...)
	if err != nil {
		log.Fatalln(err)
	}
}

func renderHtml(w http.ResponseWriter, tmpl string, vars interface{}) {
        err := html.ExecuteTemplate(w, tmpl+".gohtml", vars)
        if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
        }
}

