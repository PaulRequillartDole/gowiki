package main

import (
    "html/template"
    "os"
	"net/http"
	"log"
	"regexp"
	"errors"
	"path/filepath"
)

var templates = template.Must(template.ParseFiles("base.tmpl","edit.html", "view.html", "home.tmpl"))
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

type Page struct {
    Title string
    Body  []byte
}


func (p *Page) save() error {
    filename := p.Title + ".txt"
    return os.WriteFile("pages/"+ filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
    filename := title + ".txt"
    body, err := os.ReadFile("pages/"+ filename)
	if err != nil {
		return nil, err
	}
    return &Page{Title: title, Body: body}, nil
}

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
    m := validPath.FindStringSubmatch(r.URL.Path)
    if m == nil {
        http.NotFound(w, r)
        return "", errors.New("invalid Page Title")
    }
    return m[2], nil // The title is the second subexpression.
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page)  {
	files := []string{
        "./base.tmpl",
        "./"+tmpl+".tmpl",
    }

    // Use the template.ParseFiles() function to read the files and store the
    // templates in a template set. Notice that we can pass the slice of file
    // paths as a variadic parameter?
    ts, err := template.ParseFiles(files...)
    if err != nil {
        log.Print(err.Error())
        http.Error(w, "Internal Server Error", 500)
        return
    }

    // Use the ExecuteTemplate() method to write the content of the "base" 
    // template as the response body.
    err = ts.ExecuteTemplate(w, "base", p)
    if err != nil {
        log.Print(err.Error())
        http.Error(w, "Internal Server Error", 500)
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

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
    p, err := loadPage(title)
    if err != nil {
        p = &Page{Title: title}
    }
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
    body := r.FormValue("body")
    p := &Page{Title: title, Body: []byte(body)}
    err := p.save()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func makeHandler(fn func (http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        m := validPath.FindStringSubmatch(r.URL.Path)
        if m == nil {
            http.NotFound(w, r)
            return
        }
        fn(w, r, m[2])
    }
}

func homeHandler(w http.ResponseWriter, r *http.Request)  {

	pwd,_ := os.Getwd()
	f, err := os.Open(pwd+"/pages")
    if err != nil {
        return
    }
    files, err := f.Readdir(0)
    if err != nil {
        return
    }

	var data []*Page
    for _, v := range files {
		title := fileNameWithoutExtSliceNotation(v.Name());
		p, _ := loadPage(title)
		data = append(data, p)
    }

	if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }

    // Initialize a slice containing the paths to the two files. It's important
    // to note that the file containing our base template must be the *first*
    // file in the slice.
    tmpl := []string{
        "./base.tmpl",
        "./home.tmpl",
    }

    // Use the template.ParseFiles() function to read the files and store the
    // templates in a template set. Notice that we can pass the slice of file
    // paths as a variadic parameter?
    ts, err := template.ParseFiles(tmpl...)
    if err != nil {
        log.Print(err.Error())
        http.Error(w, "Internal Server Error", 500)
        return
    }

    // Use the ExecuteTemplate() method to write the content of the "base" 
    // template as the response body.
    err = ts.ExecuteTemplate(w, "base", data)
    if err != nil {
        log.Print(err.Error())
        http.Error(w, "Internal Server Error", 500)
    }
}

func fileNameWithoutExtSliceNotation(fileName string) string {
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}


func main()  {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
    http.HandleFunc("/edit/", makeHandler(editHandler))
    http.HandleFunc("/save/", makeHandler(saveHandler))

    log.Fatal(http.ListenAndServe(":8080", nil))
}