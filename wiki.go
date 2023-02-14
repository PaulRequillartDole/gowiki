package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var validPath = regexp.MustCompile("^/(edit|save|view|delete)/([a-zA-Z0-9-_]+)$")

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return os.WriteFile("pages/"+filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := os.ReadFile("pages/" + filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	var templates = template.Must(template.New(tmpl).Funcs(template.FuncMap{
		"safeHTML": func(b []byte) template.HTML {
			return template.HTML(b)
		},
	}).ParseFiles("./templates/_base.html", "./templates/"+tmpl+".html"))

	err := templates.ExecuteTemplate(w, "base", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}

	renderTemplate(w, "view", p)
}

func newHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		var templates = template.Must(template.New("new").Funcs(template.FuncMap{
			"safeHTML": func(b []byte) template.HTML {
				return template.HTML(b)
			},
		}).ParseFiles("./templates/_base.html", "./templates/new.html"))
		err := templates.ExecuteTemplate(w, "base", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		title := r.FormValue("title")
		title = strings.ReplaceAll(title, " ", "_")
		body := r.FormValue("body")
		p := &Page{Title: title, Body: []byte(body)}
		err := p.save()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/view/"+title, http.StatusFound)
	}
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func deleteHandler(w http.ResponseWriter, r *http.Request, title string) {
	filename := title + ".txt"
	e := os.Remove("pages/" + filename)
	if e != nil {
		log.Fatal(e)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	title = strings.ReplaceAll(title, " ", "_")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	files := checkExt(".txt")

	var data []*Page
	for _, v := range files {
		title := fileNameWithoutExtSliceNotation(v)
		p, _ := loadPage(title)
		data = append(data, p)
	}

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	var templates = template.Must(template.New("home").Funcs(template.FuncMap{
		"safeHTML": func(b []byte) template.HTML {
			return template.HTML(b)
		},
	}).ParseFiles("./templates/_base.html", "./templates/home.html"))
	err := templates.ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func fileNameWithoutExtSliceNotation(fileName string) string {
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}

func checkExt(ext string) []string {
	pathS, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	var files []string
	err = filepath.Walk(pathS, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			r, err := regexp.MatchString(ext, f.Name())
			if err == nil && r {
				files = append(files, f.Name())
			}
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return files
}

func main() {
	log.Println("Server started on: http://localhost:8080")
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/delete/", makeHandler(deleteHandler))
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/new", newHandler)
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
