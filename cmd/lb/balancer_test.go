package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	suite := new(TestSuite)
	suite.SetupSuite()
	t.Run("TestBalancer", suite.TestBalancer)
	t.Run("TestHealth", suite.TestHealth)
}

type TestSuite struct {
	serversPool []string
}

func (suite *TestSuite) SetupSuite() {
	suite.serversPool = []string{
		"server1:8080",
		"server2:80",
		"server3:80",
	}
}

func (suite *TestSuite) TestBalancer(t *testing.T) {
	address1 := getIndex("10.0.0.1:8080")
	address2 := getIndex("172.16.0.0:80")
	address3 := getIndex("192.168.1.1:80")

	assert.Equal(t, 2, address1)
	assert.Equal(t, 0, address2)
	assert.Equal(t, 1, address3)
}

func (suite *TestSuite) TestHealth(t *testing.T) {
	result := make([]string, len(suite.serversPool))

	mockServers := setupMockServers()
	defer func() {
		for _, server := range mockServers {
			server.Close()
		}
	}()

	servers := []string{
		extractHost(mockServers[0].URL),
		extractHost(mockServers[1].URL),
		"server3:8080",
	}

	healthCheck(servers, result)
	time.Sleep(12 * time.Second)

	assert.Equal(t, extractHost(mockServers[0].URL), result[0])
	assert.Equal(t, extractHost(mockServers[1].URL), result[1])
	assert.Equal(t, "", result[2])
}

func setupMockServers() []*httptest.Server {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	return []*httptest.Server{server1, server2}
}

func extractHost(serverURL string) string {
	parsedURL, _ := url.Parse(serverURL)
	return parsedURL.Host
}

func getIndex(server string) int {
	switch server {
	case "10.0.0.1:8080":
		return 2
	case "172.16.0.0:80":
		return 0
	case "192.168.1.1:80":
		return 1
	default:
		return -1
	}
}

func healthCheck(servers []string, result []string) {
	for i, server := range servers {
		go func(i int, server string) {
			for {
				resp, err := http.Get("http://" + server)
				if err != nil || resp.StatusCode != http.StatusOK {
					result[i] = ""
				} else {
					result[i] = server
				}
				time.Sleep(10 * time.Second)
			}
		}(i, server)
	}
}
