package main

import (
	"dnsmforwarder/http_helpers"
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
		http_helpers.RetError400(w, r, err.Error())
		return
	}
	http_helpers.LogMsg(r, fmt.Sprintf("decoded object: %s", obj))

	u, err := netip.ParseAddrPort(obj.Upstream)
	if err != nil {
		http_helpers.RetError400(w, r, err.Error())
		return
	}

	nsUpstreams.Store(u, true)
	resp := UpstreamModel{Upstream: u.String()}
	http_helpers.LogMsg(r, fmt.Sprintf("returning response: %s", resp))

	http_helpers.RetJSON(w, r, resp, http.StatusOK)
}

func getUpstreamsHandler(w http.ResponseWriter, r *http.Request) {
	oc := nsUpstreams.Clone()
	resp := make([]UpstreamModel, 0, len(oc))
	for k := range oc {
		resp = append(resp, UpstreamModel{Upstream: k.String()})
	}

	http_helpers.RetJSON(w, r, resp, http.StatusOK)
}

func deleteUpstreamHandler(w http.ResponseWriter, r *http.Request) {
	u, err := netip.ParseAddrPort(chi.URLParam(r, "upstream"))
	if err != nil {
		http_helpers.RetError400(w, r, err.Error())
		return
	}
	http_helpers.LogMsg(r, fmt.Sprintf("decoded params: upstream=%s", u))

	// don't delete from cache, this upstream can appear soon
	_, ok := nsUpstreams.LoadAndDelete(u)
	if ok {
		resp := UpstreamModel{Upstream: u.String()}
		http_helpers.LogMsg(r, fmt.Sprintf("returning response: %s", resp))

		http_helpers.RetJSON(w, r, resp, http.StatusOK)
	} else {
		http_helpers.RetError(w, r, "no such upstream: "+u.String(), http.StatusNotFound)
	}
}

func putOriginHandler(w http.ResponseWriter, r *http.Request) {
	var obj OriginModel
	err := json.NewDecoder(r.Body).Decode(&obj)
	if err != nil {
		http_helpers.RetError400(w, r, err.Error())
		return
	}
	http_helpers.LogMsg(r, fmt.Sprintf("decoded object: %s", obj))

	o, err := netip.ParseAddr(obj.IP)
	if err != nil {
		http_helpers.RetError400(w, r, err.Error())
		return
	}
	// we don't know what address format will be in dns RemoteAddr and api
	// so convert v4in6 format to v4
	o = CanonAddr(o)

	u, err := netip.ParseAddrPort(obj.Upstream)
	if err != nil {
		http_helpers.RetError400(w, r, err.Error())
		return
	}

	originsToNS.Store(o, u)

	resp := OriginModel{IP: o.String(), Upstream: u.String()}
	http_helpers.LogMsg(r, fmt.Sprintf("returning response: %s", resp))

	http_helpers.RetJSON(w, r, resp, http.StatusOK)
}

func getOriginsHandler(w http.ResponseWriter, r *http.Request) {
	oc := originsToNS.Clone()
	resp := make([]OriginModel, 0, len(oc))
	for k, v := range oc {
		resp = append(resp, OriginModel{IP: k.String(), Upstream: v.String()})
	}

	http_helpers.RetJSON(w, r, resp, http.StatusOK)
}

func deleteOriginHandler(w http.ResponseWriter, r *http.Request) {
	o, err := netip.ParseAddr(chi.URLParam(r, "origin"))
	if err != nil {
		http_helpers.RetError400(w, r, err.Error())
		return
	}
	http_helpers.LogMsg(r, fmt.Sprintf("decoded params: origin=%s", o))

	u, ok := originsToNS.LoadAndDelete(o)
	if ok {
		resp := OriginModel{IP: o.String(), Upstream: u.String()}
		http_helpers.LogMsg(r, fmt.Sprintf("returning response: %s", resp))

		http_helpers.RetJSON(w, r, resp, http.StatusOK)
	} else {
		http_helpers.RetError(w, r, "no such origin: "+o.String(), http.StatusNotFound)
	}
}
