package integration

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

//test

func sendRequest(baseAddress string, responseSize int, client *http.Client) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/some-data", baseAddress), nil)
	if err != nil {
		log.Printf("error creating request: %s", err)
		return nil, err
	}
	req.Header.Set("Response-Size", strconv.Itoa(responseSize))

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error: %s", err)
		return nil, err
	}

	log.Printf("response %d", resp.StatusCode)
	return resp, nil
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	suite := new(IntegrationTestSuite)
	suite.Run(t)
}

type IntegrationTestSuite struct {
	t *testing.T
}

func (s *IntegrationTestSuite) Run(t *testing.T) {
	s.t = t
	responseSize := 0
	serverNum := [6]int{1, 2, 3, 1, 3, 2}
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			responseSize = 1000
		} else {
			responseSize = 2000
		}
		server, _ := sendRequest(baseAddress, responseSize, &client)
		assert.Equal(s.t, fmt.Sprintf("server%d:8080", serverNum[i]), server.Header.Get("lb-from"))
	}
}

func (s *IntegrationTestSuite) BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}

	for i := 0; i < b.N; i++ {
		_, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		assert.NoError(s.t, err)
	}
}
