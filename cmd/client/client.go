package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var target = flag.String("target", "http://localhost:8090", "request target")

func main() {
	flag.Parse()
	client := &http.Client{Timeout: 10 * time.Second}

	endpoints := []string{
		"/api/v1/server1",
		"/api/v2/server2",
		"/api/v3/server3",
	}

	for range time.Tick(1 * time.Second) {
		for _, endpoint := range endpoints {
			resp, err := client.Get(fmt.Sprintf("%s%s", *target, endpoint))
			if err == nil {
				log.Printf("response from %s: %d", endpoint, resp.StatusCode)
			} else {
				log.Printf("error from %s: %s", endpoint, err)
			}
			if resp != nil {
				err := resp.Body.Close()
				if err != nil {
					return
				}
			}
		}
	}
}
