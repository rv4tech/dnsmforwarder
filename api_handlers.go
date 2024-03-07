package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/go-chi/chi/v5"
)

type OriginModel struct {
	IP       string `json:"ip"`
	Upstream string `json:"upstream"`
}

type UpstreamModel struct {
	Upstream string `json:"upstream"`
}

func putUpstreamHandler(w http.ResponseWriter, r *http.Request) {
	var obj UpstreamModel
	err := json.NewDecoder(r.Body).Decode(&obj)
	if err != nil {
		textError400(w, r, err.Error())
		return
	}
	logMsg(r, fmt.Sprintf("decoded object: %s", obj))

	u, err := netip.ParseAddrPort(obj.Upstream)
	if err != nil {
		textError400(w, r, err.Error())
		return
	}

	Upstreams.Store(u, true)
	resp := UpstreamModel{Upstream: u.String()}
	logMsg(r, fmt.Sprintf("returning response: %s", resp))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		textError500(w, r, err.Error())
	}
}

func getUpstreamsHandler(w http.ResponseWriter, r *http.Request) {
	oc := Upstreams.Clone()
	resp := make([]UpstreamModel, 0, len(oc))
	for k := range oc {
		resp = append(resp, UpstreamModel{Upstream: k.String()})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		textError500(w, r, err.Error())
	}
}

func deleteUpstreamHandler(w http.ResponseWriter, r *http.Request) {
	u, err := netip.ParseAddrPort(chi.URLParam(r, "upstream"))
	if err != nil {
		textError400(w, r, err.Error())
		return
	}
	logMsg(r, fmt.Sprintf("decoded params: upstream=%s", u))

	_, ok := Upstreams.LoadAndDelete(u)
	if ok {
		resp := UpstreamModel{Upstream: u.String()}
		logMsg(r, fmt.Sprintf("returning response: %s", resp))

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			textError500(w, r, err.Error())
		}
	} else {
		textError(w, r, "no such upstream: "+u.String(), http.StatusNotFound)
	}
}

func putOriginHandler(w http.ResponseWriter, r *http.Request) {
	var obj OriginModel
	err := json.NewDecoder(r.Body).Decode(&obj)
	if err != nil {
		textError400(w, r, err.Error())
		return
	}
	logMsg(r, fmt.Sprintf("decoded object: %s", obj))

	o, err := netip.ParseAddr(obj.IP)
	if err != nil {
		textError400(w, r, err.Error())
		return
	}
	// we don't know what address format will be in dns RemoteAddr and api
	// so convert v4in6 format to v4
	o = CanonAddr(o)

	u, err := netip.ParseAddrPort(obj.Upstream)
	if err != nil {
		textError400(w, r, err.Error())
		return
	}

	Origins.Store(o, u)

	resp := OriginModel{IP: o.String(), Upstream: u.String()}
	logMsg(r, fmt.Sprintf("returning response: %s", resp))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		textError500(w, r, err.Error())
	}
}

func getOriginsHandler(w http.ResponseWriter, r *http.Request) {
	oc := Origins.Clone()
	resp := make([]OriginModel, 0, len(oc))
	for k, v := range oc {
		resp = append(resp, OriginModel{IP: k.String(), Upstream: v.String()})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		textError500(w, r, err.Error())
	}
}

func deleteOriginHandler(w http.ResponseWriter, r *http.Request) {
	o, err := netip.ParseAddr(chi.URLParam(r, "origin"))
	if err != nil {
		textError400(w, r, err.Error())
		return
	}
	logMsg(r, fmt.Sprintf("decoded params: origin=%s", o))

	u, ok := Origins.LoadAndDelete(o)
	if ok {
		resp := OriginModel{IP: o.String(), Upstream: u.String()}
		logMsg(r, fmt.Sprintf("returning response: %s", resp))

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			textError500(w, r, err.Error())
		}
	} else {
		textError(w, r, "no such origin: "+o.String(), http.StatusNotFound)
	}
}
