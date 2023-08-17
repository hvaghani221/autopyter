package code

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (c *Code) PageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("plugin").ParseFS(c.fs, "templates/index.html"))
	if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		log.Println(err)
	}
}

func (c *Code) ClipHandler(w http.ResponseWriter, req *http.Request) {
	tmpl := template.Must(template.New("clip").ParseFS(c.fs, "templates/clip.html"))

	list := c.history.List()
	last := req.URL.Query().Get("start")
	lastID := int64(-1)

	if last != "" {
		var err error
		lastID, err = strconv.ParseInt(last, 10, 64)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		i := 0
		for ; i < len(list); i++ {
			if list[i].ID > lastID {
				break
			}
		}
		list = list[i:]
	}

	if len(list) > 0 {
		lastID = list[len(list)-1].ID
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
	codeID, ok := mux.Vars(req)["ID"]
	if !ok {
		http.NotFound(w, req)
		return
	}
	id, err := strconv.ParseInt(codeID, 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 400)
		return
	}
	if !c.history.Remove(id) {
		http.NotFound(w, req)
		return
	}

	w.WriteHeader(http.StatusOK)
}
