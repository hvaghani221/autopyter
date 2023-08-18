package code

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	codeID, ok := mux.Vars(r)["ID"]

	if !ok {
		http.NotFound(w, r)
		return 0, false
	}

	id, err := strconv.ParseInt(codeID, 10, 64)
	if err != nil {
		log.Println("parsing ID: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 0, false
	}
	return id, true
}

type withID interface {
	GetID() int64
}

func filterList[T withID](w http.ResponseWriter, r *http.Request, list []T) ([]T, int64, bool) {
	last := r.URL.Query().Get("start")
	lastID := int64(-1)

	if last != "" {
		var err error
		lastID, err = strconv.ParseInt(last, 10, 64)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil, 0, false
		}
		i := 0
		for ; i < len(list); i++ {
			if list[i].GetID() > lastID {
				break
			}
		}
		list = list[i:]
	}

	if len(list) > 0 {
		lastID = list[len(list)-1].GetID()
	}

	return list, lastID, true
}
