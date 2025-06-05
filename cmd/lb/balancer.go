package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/roman-mazur/architecture-practice-4-template/httptools"
	"github.com/roman-mazur/architecture-practice-4-template/signal"
)

/* ----------------------------- CLI-прапорці ----------------------------- */

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPS")
	trace      = flag.Bool("trace", false, "include tracing info into responses")
)

/* -------------------------- Модель бекенда --------------------------- */

type backend struct {
	addr    string       // host:port
	healthy atomic.Bool  // актуальний стан /health
	connCnt atomic.Int32 // активні запити (Least-Connections)
}

var (
	timeout  time.Duration
	backends []*backend
)

/* --------------------------- Дрібні хелпери -------------------------- */

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(addr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s://%s/health", scheme(), addr), nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

/* ---------------------- Least-Connections core ----------------------- */

func pickBackend() *backend {
	var best *backend
	min := int32(math.MaxInt32)

	for _, be := range backends {
		if !be.healthy.Load() {
			continue
		}
		if c := be.connCnt.Load(); c < min {
			min, best = c, be
		}
	}
	return best
}

/* ------------------------- Проксі й форвардинг ----------------------- */

func reverseProxy(be *backend) *httputil.ReverseProxy {
	p := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: scheme(),
		Host:   be.addr,
	})
	// власна Transport, щоб не використати proxy з оточення
	p.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			KeepAlive: 60 * time.Second,
			Timeout:   timeout,
		}).DialContext,
		DisableKeepAlives: false,
	}
	p.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
	}
	return p
}

func forward(be *backend, rw http.ResponseWriter, r *http.Request) {
	be.connCnt.Add(1)
	defer be.connCnt.Add(-1)

	proxy := reverseProxy(be)
	if *trace {
		rw.Header().Set("lb-from", be.addr)
	}
	proxy.ServeHTTP(rw, r)
}

/* ------------------------- entry-point / main ------------------------ */

func main() {
	flag.Parse()
	timeout = time.Duration(*timeoutSec) * time.Second

	// 1. Формуємо пул із трьох бекендів (адреси взяті з docker-compose)
	addrs := []string{"server1:8080", "server2:8080", "server3:8080"}
	for _, a := range addrs {
		be := &backend{addr: a}
		be.healthy.Store(true) // на старті вважаємо, що живий
		backends = append(backends, be)
	}

	// 2. Паралельний health-checker
	for _, be := range backends {
		be := be
		go func() {
			for range time.Tick(10 * time.Second) {
				be.healthy.Store(health(be.addr))
				log.Println(be.addr, "healthy:", be.healthy.Load(),
					"active:", be.connCnt.Load())
			}
		}()
	}

	// 3. HTTP-сервер балансувальника
	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		be := pickBackend()
		if be == nil {
			http.Error(rw, "no healthy backends", http.StatusServiceUnavailable)
			return
		}
		forward(be, rw, r)
	})

	frontend := httptools.CreateServer(*port, handler)

	log.Printf("Starting LB on :%d (trace=%v)", *port, *trace)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
