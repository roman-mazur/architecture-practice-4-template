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
	client := new(http.Client)
	client.Timeout = 10 * time.Second

	counter := 0
	for range time.Tick(1 * time.Second) {
		if (counter + 1) == 4 {
			counter = 0
		}
		counter++
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data%d", *target, counter))
		if err == nil {
			log.Printf("response %d", resp.StatusCode)
		} else {
			log.Printf("error %s", err)
		}
	}
}
