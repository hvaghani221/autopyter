package code

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func (c *Code) CodeHandler(w http.ResponseWriter, r *http.Request) {
	list := c.state.ListStates()
	last := r.URL.Query().Get("start")
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
		States []ExecutionState
		LastID int64
	}{
		States: list,
		LastID: lastID,
	}

	tmpl := template.Must(template.New("code").ParseFS(c.fs, "templates/code.html"))
	if err := tmpl.ExecuteTemplate(w, "code.html", data); err != nil {
		log.Println(err)
	}
}

func (c *Code) CodeDeleteHandler(w http.ResponseWriter, r *http.Request) {
	codeID, ok := mux.Vars(r)["ID"]
	if !ok {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.ParseInt(codeID, 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !c.state.RemoveState(id) {
		http.NotFound(w, r)
		return
	}
}

func (c *Code) ExecuteHandler(w http.ResponseWriter, r *http.Request) {
	codeID, ok := mux.Vars(r)["ID"]

	if !ok {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.ParseInt(codeID, 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	code, ok := c.history.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	c.history.Remove(id)
	if err := c.state.Execute(code.Code); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "codeExecuted")
}

func (c *Code) ResultHandler(w http.ResponseWriter, r *http.Request) {
	codeID, ok := mux.Vars(r)["ID"]

	if !ok {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.ParseInt(codeID, 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	state := c.state.GetState(id)
	if state == nil {
		http.NotFound(w, r)
		return
	}
	state.WaitForResult()

	if state.Error != nil {
		log.Println(state.Error)
		http.Error(w, state.Error.Error(), http.StatusInternalServerError)
		return
	}

	funcs := template.FuncMap{
		"gethtml": func(mimetype string, value any) template.HTML {
			if strings.HasPrefix(mimetype, "text/") {
				return template.HTML(fmt.Sprintf("<p>%s</p>", value))
			}
			if strings.HasPrefix(mimetype, "image/") {
				return template.HTML(fmt.Sprintf(`<img src="data:%s;base64,%s" alt="Embedded Image">`, mimetype, value))
			}
			return template.HTML(fmt.Sprintf(`<p>Unknown mimetype: %s</p>\n<p>%s</p>`, template.HTMLEscapeString(mimetype), template.HTMLEscapeString(fmt.Sprint(value))))
		},
	}

	tmpl := template.Must(template.New("result").Funcs(funcs).ParseFS(c.fs, "templates/result.html"))
	if err := tmpl.ExecuteTemplate(w, "result.html", state); err != nil {
		log.Println(err)
	}
}
