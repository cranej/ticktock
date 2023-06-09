package server

import (
	"embed"
	"encoding/json"
	"github.com/cranej/ticktock/store"
	"github.com/cranej/ticktock/version"
	"github.com/cranej/ticktock/view"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed asset
var asset embed.FS

func assetPath(file string) string {
	return path.Join("asset", file)
}

const (
	contentTypeHeader = "Content-Type"
	contentPng        = "image/png"
	contentHtml       = "text/html"
	contentJs         = "text/javascript"
	contentJson       = "application/json"
	contentCss        = "text/css"
	contentBinary     = "application/octet-stream"
)

type Env struct {
	Store store.Store
}

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	data, err := asset.ReadFile(assetPath("index.html"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set(contentTypeHeader, contentHtml)
	w.Write(data)
}

func anyFile(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	file := p.ByName("filepath")
	data, err := asset.ReadFile(assetPath(file))

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch path.Ext(file) {
	case ".png":
		w.Header().Add(contentTypeHeader, contentPng)
	case ".js":
		w.Header().Add(contentTypeHeader, contentJs)
	case ".css":
		w.Header().Add(contentTypeHeader, contentCss)
	default:
		w.Header().Add(contentTypeHeader, contentBinary)
	}

	w.Write(data)
}

func (env *Env) apiRecent(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	titles, err := env.Store.RecentTitles(5)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJson(w, titles)
}

func (env *Env) apiLastActivity(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	title := ps.ByName("title")
	last, err := env.Store.LastClosed(title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if last == nil {
		http.Error(w, "No such activity.", http.StatusNotFound)
		return
	}

	t, err := template.New("activity").Parse(`<h2>{{.Title}}</h2>
	<h3>{{.Start.Local.Format "2006-01-02 15:04:05"}} ~ {{.End.Local.Format "2006-01-02 15:04:05"}}</h3>
	<pre>{{.Notes}}</pre>`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var out strings.Builder
	if err := t.Execute(&out, last); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, out.String())
}

func (env *Env) apiOngoing(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	activity, err := env.Store.Ongoing()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJson(w, activity)
}

func (env *Env) apiStart(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	title := ps.ByName("title")

	if err := env.Store.StartTitle(title, ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (env *Env) apiCloseActivity(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	notes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = env.Store.CloseActivity(string(notes))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (env *Env) apiReport(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	viewType := r.Form.Get("view_type")
	if viewType == "" {
		viewType = "summary"
	}
	start, end := ps.ByName("start"), ps.ByName("end")

	startTime, err := time.ParseInLocation(time.DateOnly, start, time.Local)
	if err != nil {
		http.Error(w, "Invalid format of query start", http.StatusBadRequest)
		return
	}

	endTime, err := time.ParseInLocation(time.DateOnly, end, time.Local)
	if err != nil {
		http.Error(w, "Invalid format of query end", http.StatusBadRequest)
		return
	}

	startTime, endTime = setTimeAndUTC(startTime, 0, 0, 0), setTimeAndUTC(endTime, 23, 59, 59)
	activities, err := env.Store.Closed(startTime, endTime, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	view, err := view.Render(activities, viewType, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	io.WriteString(w, view)
}

func (env *Env) Run(addr string) error {
	router := httprouter.New()
	router.GET("/", index)
	router.GET("/static/*filepath", anyFile)
	router.GET("/api/recent", env.apiRecent)
	router.GET("/api/latest/:title", env.apiLastActivity)
	router.GET("/api/ongoing", env.apiOngoing)
	router.POST("/api/start/:title", env.apiStart)
	router.POST("/api/finish", env.apiCloseActivity)
	router.GET("/api/report/:start/:end", env.apiReport)
	router.GET("/version", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		io.WriteString(w, version.Version)
	})

	log.Println("Serve at:", addr)
	return http.ListenAndServe(addr, router)
}

func writeJson(w http.ResponseWriter, v any) {
	j, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(contentTypeHeader, contentJson)
	w.Write(j)
}

func setTimeAndUTC(t time.Time, hour, minutes, seconds int) time.Time {
	tt := time.Date(t.Year(), t.Month(), t.Day(),
		hour, minutes, seconds, 0, time.Local)
	return tt.UTC()
}
