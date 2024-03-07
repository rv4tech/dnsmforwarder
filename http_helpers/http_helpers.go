package http_helpers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/gofrs/uuid/v5"
)

var ErrorMsgKey = "error"

func LogMsg(r *http.Request, msg string) {
	log.Printf("[%s] handler %v %v: %v",
		middleware.GetReqID(r.Context()),
		r.Method,
		r.URL.Path,
		msg,
	)
}

func RetJSON(w http.ResponseWriter, r *http.Request, data any, code int) {
	w.Header().Set(middleware.RequestIDHeader, middleware.GetReqID(r.Context()))

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	if b, err := json.Marshal(data); err == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(code)
		_, err := w.Write(b)
		if err != nil {
			LogMsg(r, fmt.Sprintf("write body error: %v", err))
			// this will be recovered via Recoverer middleware
			panic(err)
		}
	} else {
		RetError500(w, r, err.Error())
	}
}

func RetError(w http.ResponseWriter, r *http.Request, data any, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)

	b := jsonMarshalSilent(data)

	_, err := w.Write(fmt.Appendf(nil, `{"%s": %s}`, ErrorMsgKey, b))
	if err != nil {
		LogMsg(r, fmt.Sprintf("write body error: %v", err))
		panic(err)
	}
	LogMsg(r, fmt.Sprintf("error: %v: %v", http.StatusText(code), data))
}

// typical error for bad input data, parsing errors, etc
func RetError400(w http.ResponseWriter, r *http.Request, data any) {
	RetError(w, r, data, http.StatusBadRequest)
}

// typical error for something, that prevents handler to complete successfully
func RetError500(w http.ResponseWriter, r *http.Request, data any) {
	RetError(w, r, data, http.StatusInternalServerError)
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	// json format for consistency
	w.Write([]byte(`{"` + ErrorMsgKey + `": "404 page not found"}`))
	// don't log 404, don't panic
}

func RequestID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestID := r.Header.Get(middleware.RequestIDHeader)
		if requestID == "" {
			// slower than default chi atomic implementation, but it is uuid =)
			u := uuid.Must(uuid.NewV7())
			requestID = u.String()
		}
		ctx = context.WithValue(ctx, middleware.RequestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func jsonMarshalSilent(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(fmt.Sprintf(`"marshal_err_base64: %s"`, base64.StdEncoding.EncodeToString([]byte(err.Error()))))
	}
	return b
}
