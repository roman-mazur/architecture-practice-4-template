package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/roman-mazur/design-practice-2-template/httptools"
	"github.com/roman-mazur/design-practice-2-template/signal"
)

var (
	port         = flag.Int("port", 8090, "load balancer port")
	timeoutSec   = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https        = flag.Bool("https", false, "whether backends support HTTPs")
	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

type Server struct {
	URL     string
	ConnCnt int
}

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []*Server{
		{URL: "server1:8080"},
		{URL: "server2:8080"},
		{URL: "server3:8080"},
	}
	mutex sync.Mutex
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(s *Server) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), s.URL), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func minServerIndex() int {
	minIndex := 0
	minConnCnt := serversPool[0].ConnCnt

	for i, server := range serversPool {
		if server.ConnCnt < minConnCnt {
			minIndex = i
			minConnCnt = server.ConnCnt
		}
	}

	return minIndex
}

func forward(rw http.ResponseWriter, r *http.Request) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)

	mutex.Lock()
	minIndex := minServerIndex()
	dst := serversPool[minIndex]
	dst.ConnCnt++
	mutex.Unlock()

	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst.URL
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst.URL

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst.URL)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst.URL, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

func main() {
	flag.Parse()

	for _, server := range serversPool {
		go func(s *Server) {
			for range time.Tick(10 * time.Second) {
				mutex.Lock()
				log.Printf("%s: health=%t, connCnt=%d", s.URL, health(s), s.ConnCnt)
				mutex.Unlock()
			}
		}(server)
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		forward(rw, r)
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
