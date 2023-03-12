package server

import (
	"cranejin.com/ticktock/store"
	"embed"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
	"path"
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

func static(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	data, err := asset.ReadFile(assetPath("index.html"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set(contentTypeHeader, contentHtml)
	w.Write(data)
}

func AnyFile(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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

func (env *Env) ApiRecent(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	titles, err := env.Store.RecentTitles(5)

	if err != nil {
		writeError(w, err)
		return
	}

	writeJson(w, titles)
}

func (env *Env) ApiLatest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	title := ps.ByName("title")
	last, err := env.Store.LastFinished(title)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJson(w, last)
}

func (env *Env) ApiUnfinished(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	entry, err := env.Store.Ongoing()
	if err != nil {
		writeError(w, err)
		return
	}

	writeJson(w, entry)
}

func (env *Env) ApiStart(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	title := ps.ByName("title")

	if err := env.Store.StartTitle(title, ""); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (env *Env) ApiFinish(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// title ignored
	notes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, err)
		return
	}

	_, err = env.Store.Finish(string(notes))

	if err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (env *Env) ApiReport(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := r.ParseForm(); err != nil {
		writeError(w, err)
		return
	}
	viewType := r.Form.Get("view_type")
	if viewType == "" {
		viewType = "summary"
	}
	start, end := ps.ByName("start"), ps.ByName("end")

	startTime, err := time.ParseInLocation(time.DateOnly, start, time.Local)
	if err != nil {
		writeBadRequest(w, "Invalid format of query start")
		return
	}

	endTime, err := time.ParseInLocation(time.DateOnly, end, time.Local)
	if err != nil {
		writeBadRequest(w, "Invalid format of query end")
		return
	}

	startTime, endTime = startTime.UTC(), endTime.UTC()
	entries, err := env.Store.Finished(startTime, endTime)
	if err != nil {
		writeError(w, err)
		return
	}

	switch viewType {
	case "summary":
		summary := store.NewSummary(entries)
		io.WriteString(w, summary.String())
	case "detail":
		detail := store.NewDetail(entries)
		io.WriteString(w, detail.String())
	case "dist":
		dist := store.NewDist(entries)
		io.WriteString(w, dist.String())
	}
}

func (env *Env) Run(addr string) error {
	router := httprouter.New()
	router.GET("/", static)
	router.GET("/static/*filepath", AnyFile)
	router.GET("/api/recent", env.ApiRecent)
	router.GET("/api/latest/:title", env.ApiLatest)
	router.GET("/api/unfinished", env.ApiUnfinished)
	router.POST("/api/start/:title", env.ApiStart)
	router.POST("/api/finish/:title", env.ApiFinish)
	router.GET("/api/report-by-date/:start/:end", env.ApiReport)

	return http.ListenAndServe(addr, router)
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(w, err.Error())
}

func writeJson(w http.ResponseWriter, v any) {
	j, err := json.Marshal(v)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set(contentTypeHeader, contentJson)
	w.Write(j)
}

func writeBadRequest(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	io.WriteString(w, msg)
}
