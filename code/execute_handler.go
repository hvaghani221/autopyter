package code

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

func (c *Code) CodeHandler(w http.ResponseWriter, r *http.Request) {
	list, lastID, ok := filterList(w, r, c.state.ListStates(true))
	if !ok {
		return
	}

	data := struct {
		States []*ExecutionState
		LastID int64
		Debug  bool
	}{
		States: list,
		LastID: lastID,
		Debug:  c.debug,
	}

	tmpl := template.Must(template.New("code").ParseFS(c.fs, "templates/code.html"))
	if err := tmpl.ExecuteTemplate(w, "code.html", data); err != nil {
		log.Println(err)
	}
}

func (c *Code) CodeDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if !c.state.RemoveState(id) {
		log.Println("Could not remove state", id)
		http.NotFound(w, r)
		return
	}
}

func (c *Code) ExecuteHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
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
	id, ok := parseID(w, r)
	if !ok {
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
				return template.HTML(fmt.Sprintf("<pre>%s</pre>", value))
			}
			if strings.HasPrefix(mimetype, "image/") {
				return template.HTML(fmt.Sprintf(`<img src="data:%s;base64,%s" alt="Embedded Image">`, mimetype, value))
			}
			return template.HTML(fmt.Sprintf(`<p>Unknown mimetype: %s</p>\n<p>%s</p>`, template.HTMLEscapeString(mimetype), template.HTMLEscapeString(fmt.Sprint(value))))
		},
		"ashtml": func(input string) template.HTML {
			return template.HTML(input)
		},
	}

	tmpl := template.Must(template.New("result").Funcs(funcs).ParseFS(c.fs, "templates/result.html"))
	if err := tmpl.ExecuteTemplate(w, "result.html", state); err != nil {
		log.Println(err)
	}
}

func (c *Code) StateHander(w http.ResponseWriter, r *http.Request) {
	list, lastID, ok := filterList(w, r, c.state.ListStates(false))
	if !ok {
		return
	}

	data := struct {
		States []*ExecutionState
		LastID int64
	}{
		States: list,
		LastID: lastID,
	}

	funcs := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}

	if len(data.States) > 0 {
		w.Header().Set("HX-Trigger", "stateUpdated")
	}

	tmpl := template.Must(template.New("state").Funcs(funcs).ParseFS(c.fs, "templates/state.html"))
	if err := tmpl.ExecuteTemplate(w, "state.html", data); err != nil {
		log.Println(err)
	}
}

func (c *Code) StateDeleteHander(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if !c.state.RemovePreviousState(id) {
		http.NotFound(w, r)
		return
	}

	c.state.ResetState(true)
	w.Header().Set("HX-Trigger", "stateReset")
	w.Header().Set("HX-Refresh", "true")
}

func (c *Code) StateResetHander(w http.ResponseWriter, r *http.Request) {
	c.state.ResetState(false)
	w.Header().Set("HX-Trigger", "stateReset")
	w.Header().Set("HX-Refresh", "true")
}

func (c *Code) SelectHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := c.state.Select(id); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "stateSelected")
	// TODO: fix with custom events
	w.Header().Set("HX-Refresh", "true")
}

func (c *Code) EditorHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("editor").ParseFS(c.fs, "templates/editor.html"))
	if err := tmpl.ExecuteTemplate(w, "editor.html", nil); err != nil {
		log.Println(err)
	}
}

func (c *Code) EditorExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println("Error parsing editor execute request form", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	code := r.Form.Get("code")
	if err := c.state.Execute(code); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "codeExecuted")
	tmpl := template.Must(template.New("editor").ParseFS(c.fs, "templates/editor.html"))
	if err := tmpl.ExecuteTemplate(w, "editor.html", nil); err != nil {
		log.Println(err)
	}
}
