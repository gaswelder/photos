package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

const CacheDir = "cache"

const hello = `
  __________
< Oh, hello. >
  ----------
         \   ^__^ 
          \  (oo)\_______
             (__)\       )\/\\
                 ||----w |
                 ||     ||
    
`

//go:embed tpl.html
var tpl string

var modelTpl *template.Template

func init() {
	var err error
	modelTpl, err = template.New("model").Parse(`
	{{ if eq (len .Images) 1 }}
	<article class="model-single">
	{{ else }}
	<article>
	{{ end }}
		<h3>{{.Name}}</h3>
		<figure>
			{{ range .Images }}
				<img src="img/{{ . }}">
			{{ end }}
			 <figcaption>{{.Desc}}</figcaption>
		</figure>
	</article>
	`)
	if err != nil {
		panic(err)
	}
}

func main() {
	// Load the config.
	var albums map[string]album
	conf, err := os.Open("conf.json")
	if err != nil {
		log.Fatal(err)
	}
	dec := json.NewDecoder(conf)
	dec.DisallowUnknownFields()
	if err = dec.Decode(&albums); err != nil {
		log.Fatal(err)
	}

	limit = make(chan bool, 1)

	// Initialize all image paths.
	for k, a := range albums {
		log.Println("initializing paths for", k)
		_, err := a.entries()
		if err != nil {
			log.Fatal(err)
		}
	}

	// Index page.
	// Don't want to list all albums, showing only a placeholder.
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(hello))
	}))

	// Shows a full album.
	http.Handle("/{album}/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gname := r.PathValue("album")
		album, ok := albums[gname]
		if !ok {
			w.WriteHeader(404)
			w.Write([]byte("album not found"))
			return
		}
		models, err := album.entries()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		html := renderMain(models)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte(html))
	}))

	// Serves a small image from an album.
	http.Handle("/{album}/img/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origPath := imagePath(r.PathValue("id"))
		copyPath, err := sizeCopy(CacheDir, origPath, 300, 200)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.ServeFile(w, r, copyPath)
	}))

	// Serves a large image by its hash.
	http.Handle("/img/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := imagePath(r.PathValue("id"))
		if p == "" {
			http.NotFound(w, r)
			return
		}
		c, err := sizeCopy(CacheDir, p, 1600, 1600)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.ServeFile(w, r, c)
	}))

	log.Println("http://localhost:8001")
	// Here it's not localhost because it's to be used a container.
	log.Fatal(http.ListenAndServe(":8001", nil))
}

func renderMain(models []entry) string {
	var results []string
	for _, m := range models {
		w := bytes.NewBuffer(nil)
		err := modelTpl.Execute(w, m)
		if err != nil {
			panic(err)
		}
		results = append(results, w.String())
	}
	return tpl + strings.Join(results, "")
}
