package main

import (
	"fmt"
	"log"

	"github.com/miekg/dns"
)

func Lookup(server string, req *dns.Msg) (*dns.Msg, error) {
	// TODO: remove edns0
	// TODO: context with timeout
	response, _, err := DNSClient.Exchange(req, server)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func DNSReqHandler(w dns.ResponseWriter, req *dns.Msg) {
	var resp *dns.Msg
	var respErr error

	origin := CanonAddrFromStringSilent(w.RemoteAddr().String())
	upstream, _ := Origins.Load(CanonAddrFromStringSilent(w.RemoteAddr().String()))
	_, hasUpstream := Upstreams.Load(upstream)

	reqId := fmt.Sprintf("%v/%v/%v", req.Id, origin, upstream)
	log.Printf("[dns.reqid=%v] received", reqId)

	if !hasUpstream {
		respErr = fmt.Errorf("no upstream found: '%v'", upstream)
	} else {
		switch req.Opcode {
		case dns.OpcodeQuery, dns.OpcodeIQuery:
			resp, respErr = Lookup(upstream.String(), req)
		}
	}

	if respErr != nil {
		resp = new(dns.Msg).SetRcode(req, dns.RcodeServerFailure)
		log.Printf("[dns.reqid=%v] forward error: %v", reqId, respErr)
	} else if resp != nil {
		resp.SetReply(req)
		for _, rr := range resp.Answer {
			rr.Header().Ttl = 900
		}
	} else {
		resp = new(dns.Msg).SetRcode(req, dns.RcodeNotImplemented)
		log.Printf("[dns.reqid=%v] method not implemented: %v", reqId, dns.OpcodeToString[req.Opcode])
	}

	log.Printf("[dns.reqid=%v] req:\n%s", reqId, req)
	log.Printf("[dns.reqid=%v] resp:\n%s", reqId, resp)

	err := w.WriteMsg(resp)
	if err != nil {
		log.Printf("[dns.reqid=%v] write msg error: %v", reqId, err)
	}
	log.Printf("[dns.reqid=%v] processed", reqId)
}
