package main

import (
	"crypto/sha1"
	"dnsmforwarder/http_helpers"
	"dnsmforwarder/rwmutex_map"
	"flag"
	"log"
	"net/http"
	"net/netip"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jellydator/ttlcache/v3"
	"github.com/miekg/dns"
)

var (
	addr                 string
	dnsAddr              string
	dnsTimeout           int
	dnsRewriteTTL        int
	dnsCacheTTL          int
	dnsUDPBuffer         int
	upstreamIgnoreStatus bool
)

var (
	originsToNS *rwmutex_map.Map[netip.Addr, netip.AddrPort]
	nsUpstreams *rwmutex_map.Map[netip.AddrPort, bool]
	dnsClient   *dns.Client
	dnsCache    *ttlcache.Cache[[sha1.Size]byte, []byte]
)

func init() {
	flag.StringVar(&addr, "listen", ":10080", "addr:port to listen on")
	flag.StringVar(&dnsAddr, "listen-dns", ":10053", "addr:port to listen on")
	flag.IntVar(&dnsRewriteTTL, "rewrite-ttl", 900, "rewrite records TTL (seconds), set zero to disable")
	flag.IntVar(&dnsCacheTTL, "cache-ttl", 900, "internal cache TTL (seconds), set zero to disable")
	flag.IntVar(&dnsTimeout, "upstream-timeout", 10, "upstream dns request timeout (seconds)")
	flag.IntVar(&dnsUDPBuffer, "udp-buffer", 0, "udp buffer size for dns client")
	flag.BoolVar(&upstreamIgnoreStatus, "ignore-upstream-status", false,
		"try to send request to upstream even if it seems offline "+
			"(no PUT /upstreams was called) and return real dns "+
			"response (may be timeout) instead of ServerFailure")
	flag.Parse()
}

func main() {
	originsToNS = rwmutex_map.New[netip.Addr, netip.AddrPort]()
	nsUpstreams = rwmutex_map.New[netip.AddrPort, bool]()

	if dnsCacheTTL > 0 {
		dnsCache = ttlcache.New[[sha1.Size]byte, []byte](
			ttlcache.WithTTL[[sha1.Size]byte, []byte](time.Duration(dnsCacheTTL) * time.Second),
		)
		go dnsCache.Start()
	}

	dnsClient = new(dns.Client)
	dnsClient.Timeout = time.Duration(dnsTimeout) * time.Second
	dnsClient.Net = "udp"
	if dnsUDPBuffer > 0 {
		dnsClient.UDPSize = uint16(dnsUDPBuffer)
	}
	dns.HandleFunc(".", dnsReqHandler)

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
	r.NotFound(http_helpers.NotFound)

	r.Use(http_helpers.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.AllowContentType(
		"application/json",
		// used for r.With(AllowContentType(...)).Patch():
		// "application/merge-patch+json",
		// "application/json-patch+json",
	))

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
