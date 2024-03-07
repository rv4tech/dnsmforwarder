package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/miekg/dns"
)

func removeECSOption(msg *dns.Msg) {
	for i, extra := range msg.Extra {
		if opt, ok := extra.(*dns.OPT); ok {
			for j := len(opt.Option) - 1; j >= 0; j-- {
				if _, ok := opt.Option[j].(*dns.EDNS0_SUBNET); ok {
					// Remove the ECS option by filtering it out
					opt.Option = append(opt.Option[:j], opt.Option[j+1:]...)
				}
			}
			// If there are no more options left in the OPT record, remove the OPT record itself
			if len(opt.Option) == 0 {
				msg.Extra = append(msg.Extra[:i], msg.Extra[i+1:]...)
			}
		}
	}
}

func lookup(server string, req *dns.Msg) (*dns.Msg, bool, error) {
	qHash := ""
	for _, q := range req.Question {
		qHash += q.String()
	}

	if dnsCache != nil {
		m := dnsCache.Get(qHash, ttlcache.WithDisableTouchOnHit[string, string]())
		if m != nil {
			resp := new(dns.Msg)
			err := resp.Unpack([]byte(m.Value()))
			if err == nil {
				return resp, true, nil
			}
		}
	}

	removeECSOption(req)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(dnsTimeout)*time.Second)
	defer cancel()
	resp, _, err := dnsClient.ExchangeContext(ctx, req, server)

	if resp != nil && err == nil && dnsCache != nil {
		m, err := resp.Pack()
		if err == nil {
			dnsCache.Set(qHash, string(m), 0)
		}
	}
	return resp, false, err
}

func dnsReqHandler(w dns.ResponseWriter, req *dns.Msg) {
	var resp *dns.Msg
	var respErr error

	origin := CanonAddrFromStringSilent(w.RemoteAddr().String())
	upstream, _ := originsToNS.Load(CanonAddrFromStringSilent(w.RemoteAddr().String()))
	_, hasUpstream := nsUpstreams.Load(upstream)

	reqId := fmt.Sprintf("%v/%v/%v", req.Id, origin, upstream)
	log.Printf("[dns.reqid=%v] received", reqId)

	if !hasUpstream {
		respErr = fmt.Errorf("no upstream found: '%v'", upstream)
	} else {
		switch req.Opcode {
		case dns.OpcodeQuery, dns.OpcodeIQuery:
			var cached bool
			resp, cached, respErr = lookup(upstream.String(), req)
			if cached {
				log.Printf("[dns.reqid=%v] got cached response", reqId)
			}
		}
	}

	if respErr != nil {
		resp = new(dns.Msg).SetRcode(req, dns.RcodeServerFailure)
		log.Printf("[dns.reqid=%v] forward error: %v", reqId, respErr)
	} else if resp != nil {
		resp.SetReply(req)
		if dnsRewriteTTL > 0 {
			for _, rr := range resp.Answer {
				rr.Header().Ttl = uint32(dnsRewriteTTL)
			}
		}
	} else {
		resp = new(dns.Msg).SetRcode(req, dns.RcodeNotImplemented)
		log.Printf("[dns.reqid=%v] method not implemented: %v", reqId, dns.OpcodeToString[req.Opcode])
	}

	if os.Getenv("DEBUG") == "1" {
		log.Printf("[dns.reqid=%v] req:\n%s", reqId, req)
		log.Printf("[dns.reqid=%v] resp:\n%s", reqId, resp)
	}

	err := w.WriteMsg(resp)
	if err != nil {
		log.Printf("[dns.reqid=%v] write msg error: %v", reqId, err)
	}
	log.Printf("[dns.reqid=%v] processed", reqId)
}
