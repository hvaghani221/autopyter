package code

import (
	"html/template"
	"log"
	"net/http"
)

func (c *Code) PageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("plugin").ParseFS(c.fs, "templates/index.html"))
	if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		log.Println(err)
	}
}

func (c *Code) ClipHandler(w http.ResponseWriter, req *http.Request) {
	tmpl := template.Must(template.New("clip").ParseFS(c.fs, "templates/clip.html"))

	list, lastID, ok := filterList(w, req, c.history.List())
	if !ok {
		return
	}

	data := struct {
		Items  []HistoryItem
		LastID int64
	}{
		Items:  list,
		LastID: lastID,
	}

	if err := tmpl.ExecuteTemplate(w, "clip.html", data); err != nil {
		log.Println(err)
	}
}

func (c *Code) ClipDeleteHandler(w http.ResponseWriter, req *http.Request) {
	id, ok := parseID(w, req)
	if !ok {
		return
	}

	if !c.history.Remove(id) {
		http.NotFound(w, req)
		return
	}
	w.WriteHeader(http.StatusOK)
}
