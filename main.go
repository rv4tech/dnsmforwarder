package main

import (
	"dnsmforwarder/rwmutex_map"
	"flag"
	"log"
	"net/http"
	"net/netip"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/miekg/dns"
)

var (
	addr    string
	dnsAddr string
)

var (
	Origins   *rwmutex_map.Map[netip.Addr, netip.AddrPort]
	Upstreams *rwmutex_map.Map[netip.AddrPort, bool]
	DNSClient *dns.Client
)

func init() {
	flag.StringVar(&addr, "listen", ":10080", "addr:port to listen on")
	flag.StringVar(&dnsAddr, "listen-dns", ":10053", "addr:port to listen on")
	flag.Parse()
}

func main() {
	Origins = rwmutex_map.New[netip.Addr, netip.AddrPort]()
	Upstreams = rwmutex_map.New[netip.AddrPort, bool]()

	DNSClient = new(dns.Client)
	// TODO: test with big req/resps coming from tcp (convert to udp+edns0?)
	DNSClient.Net = "udp"

	dns.HandleFunc(".", DNSReqHandler)

	go func() {
		// TODO: listen on tcp too
		server := &dns.Server{Addr: dnsAddr, Net: "udp"}
		log.Printf("Starting at %s\n", dnsAddr)
		err := server.ListenAndServe()
		if err != nil {
			log.Fatalf("Failed to start server: %s\n ", err.Error())
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	r.Route("/api/v1", func(r chi.Router) {
		r.Put("/origins", putOriginHandler)
		r.Get("/origins", getOriginsHandler)
		r.Delete("/origins/{origin}", deleteOriginHandler)

		r.Put("/upstreams", putUpstreamHandler)
		r.Get("/upstreams", getUpstreamsHandler)
		r.Delete("/upstreams/{upstream}", deleteUpstreamHandler)
	})

	log.Printf("Starting at %s\n", addr)
	err := http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatal(err)
	}
}
