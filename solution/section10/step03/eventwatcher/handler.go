package eventwatcher

import (
	"embed"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/tenntenn/connpass"
)

var (
	//go:embed _template/*.html
	tmplFS embed.FS
	tmpl   = template.Must(template.ParseFS(tmplFS, "_template/*.html"))
)

func (ew *EventWatcher) initHandlers() {
	ew.mux.HandleFunc("/", ew.HandleIndex)
	ew.mux.HandleFunc("/add", ew.HandleAdd)
	ew.mux.HandleFunc("/remove", ew.HandleRemove)
	ew.mux.HandleFunc("/feed", ew.HandleFeed)
}

func (ew *EventWatcher) HandleIndex(w http.ResponseWriter, r *http.Request) {
	cs, err := ew.Conditions(r.Context(), 10)
	if err != nil {
		ew.error(w, err, http.StatusInternalServerError)
		return
	}

	es, err := ew.Events(r.Context(), cs)
	if err != nil {
		ew.error(w, err, http.StatusInternalServerError)
		return
	}

	data := struct {
		Conditions []*Condition
		Events     []*connpass.Event
	}{
		Conditions: cs,
		Events:     es,
	}

	if err := tmpl.ExecuteTemplate(w, "index", data); err != nil {
		ew.error(w, err, http.StatusInternalServerError)
		return
	}
}

func (ew *EventWatcher) HandleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		err := errors.New("MethodがPOSTではありません")
		ew.error(w, err, http.StatusMethodNotAllowed)
		return
	}

	kind := r.FormValue("kind")
	if kind == "" {
		err := errors.New("種類が指定されていません")
		ew.error(w, err, http.StatusBadRequest)
		return
	}

	value := r.FormValue("value")
	if value == "" {
		err := errors.New("値が指定されていません")
		ew.error(w, err, http.StatusBadRequest)
		return
	}

	c := &Condition{
		Kind:  kind,
		Value: value,
	}

	if err := ew.AddCondition(r.Context(), c); err != nil {
		ew.error(w, err, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (ew *EventWatcher) HandleRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		err := errors.New("MethodがPOSTではありません")
		ew.error(w, err, http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		ew.error(w, err, http.StatusBadRequest)
		return
	}

	if err := ew.RemoveCondition(r.Context(), id); err != nil {
		ew.error(w, err, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (ew *EventWatcher) HandleFeed(w http.ResponseWriter, r *http.Request) {

	cs, err := ew.Conditions(r.Context(), 10)
	if err != nil {
		ew.error(w, err, http.StatusInternalServerError)
		return
	}

	es, err := ew.Events(r.Context(), cs)
	if err != nil {
		ew.error(w, err, http.StatusInternalServerError)
		return
	}

	feed := ew.Feed(es, r.URL.String())

	format := r.FormValue("format")
	switch format {
	case "atom":
		if err := feed.WriteAtom(w); err != nil {
			ew.error(w, err, http.StatusInternalServerError)
			return
		}
	case "json":
		if err := feed.WriteJSON(w); err != nil {
			ew.error(w, err, http.StatusInternalServerError)
			return
		}
	default:
		if err := feed.WriteRss(w); err != nil {
			ew.error(w, err, http.StatusInternalServerError)
			return
		}
	}
}

func (ew *EventWatcher) error(w http.ResponseWriter, err error, code int) {
	log.Println("Error:", err)
	http.Error(w, http.StatusText(code), code)
}
