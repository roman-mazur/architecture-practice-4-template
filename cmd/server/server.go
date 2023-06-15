package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Dimasenchylo/kpi-lab4/httptools"
	"github.com/Dimasenchylo/kpi-lab4/signal"
)

var port = flag.Int("port", 8080, "server port")

const confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
const confHealthFailure = "CONF_HEALTH_FAILURE"

type Req struct {
	Value string `json:"value"`
}

type Resp struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func main() {
	h := new(http.ServeMux)
	client := http.DefaultClient
	h.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "text/plain")
		if failConfig := os.Getenv(confHealthFailure); failConfig == "true" {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte("FAILURE"))
		} else {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("OK"))
		}
	})

	report := make(Report)

	h.HandleFunc("/api/v1/some-data", func(rw http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		response, err := client.Get(fmt.Sprintf("http://db:8083/db/%s", key))
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		statusOk := response.StatusCode >= 200 && response.StatusCode < 300

		if !statusOk {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		respDelayString := os.Getenv(confResponseDelaySec)
		if delaySec, parseErr := strconv.Atoi(respDelayString); parseErr == nil && delaySec > 0 && delaySec < 300 {
			time.Sleep(time.Duration(delaySec) * time.Second)
		}

		report.Process(r)

		rw.Header().Set("content-type", "application/json")
		rw.WriteHeader(http.StatusOK)

		responseSize := 1024
		if sizeHeader := r.Header.Get("Response-Size"); sizeHeader != "" {
			if size, err := strconv.Atoi(sizeHeader); err == nil && size > 0 {
				responseSize = size
			}
		}

		responseData := make([]string, responseSize)
		for i := 0; i < responseSize; i++ {
			responseData[i] = strconv.Itoa(responseSize)
		}

		_ = json.NewEncoder(rw).Encode(responseData)
	})

	h.Handle("/report", report)

	server := httptools.CreateServer(*port, h)
	server.Start()
	buff := new(bytes.Buffer)
	body := Req{Value: time.Now().Format(time.RFC3339)}
	json.NewEncoder(buff).Encode(body)
	res, err := client.Post("http://db:8083/db/team", "application/json", buff)
	if err != nil {
		fmt.Println("Failed to send POST request:", err)
		return
	}
	defer res.Body.Close()
	signal.WaitForTerminationSignal()
}
