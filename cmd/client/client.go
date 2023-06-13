package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

var target = flag.String("target", "http://localhost:8090", "request target")
var responseSize = flag.Int("size", 2023, "desired response size")

func main() {
	flag.Parse()
	client := new(http.Client)
	client.Timeout = 10 * time.Second

	for range time.Tick(1 * time.Second) {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", *target))
		resp.Header.Set("Response-Size", strconv.Itoa(*responseSize))
		if err == nil {
			log.Printf("response %d", resp.StatusCode)
		} else {
			log.Printf("error %s", err)
		}
	}
}
