package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func logMsg(r *http.Request, msg string) {
	log.Printf("[%s] http handler %s: %s",
		middleware.GetReqID(r.Context()),
		r.URL.Path,
		msg)
}

func textError(w http.ResponseWriter, r *http.Request, msg string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, msg)
	log.Printf("[%s] http handler %s error: %s: %s",
		middleware.GetReqID(r.Context()),
		r.URL.Path,
		http.StatusText(code),
		msg)
}

func textError400(w http.ResponseWriter, r *http.Request, msg string) {
	textError(w, r, msg, http.StatusBadRequest)
}

func textError500(w http.ResponseWriter, r *http.Request, msg string) {
	textError(w, r, msg, http.StatusBadRequest)
}
