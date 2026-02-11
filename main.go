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

var entryTpl *template.Template

func init() {
	var err error
	entryTpl, err = template.New("model").Parse(`
	{{ if eq (len .Images) 1 }}
	<article class="model-single">
	{{ else }}
	<article>
	{{ end }}
		<h3>{{.Name}}</h3>
		<figure>
			{{ range .Images }}
				<a href="../img/{{ . }}"><img src="img/{{ . }}" alt=""></a>
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
	var albums map[string]*album
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
		log.Println("loading", k)
		if err := a.load(); err != nil {
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
		serveAlbum(albums, w, gname, "")
	}))

	http.Handle("/{album}/{filter}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gname := r.PathValue("album")
		filter := r.PathValue("filter")
		serveAlbum(albums, w, gname, filter)
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

func serveAlbum(albums map[string]*album, w http.ResponseWriter, name, filter string) {
	album, ok := albums[name]
	if !ok {
		w.WriteHeader(404)
		w.Write([]byte("album not found"))
		return
	}
	entries, err := album.entries(filter)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	html := renderMain(entries)
	w.Header().Add("Content-Type", "text/html")
	w.Write([]byte(html))
}

func renderMain(entries []entry) string {
	var results []string
	for _, e := range entries {
		w := bytes.NewBuffer(nil)
		err := entryTpl.Execute(w, e)
		if err != nil {
			panic(err)
		}
		results = append(results, w.String())
	}
	return tpl + strings.Join(results, "")
}
