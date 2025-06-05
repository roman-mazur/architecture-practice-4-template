package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

const (
	lbAddr         = "http://balancer:8090/api/v1/some-data"
	healthEndpoint = "/health"
	requestsCount  = 15
	timeout        = 5 * time.Second
	retryInterval  = 20 * time.Second
	maxRetries     = 30
)

func TestLeastConnectionsDistribution(t *testing.T) {
	// Чекаємо, поки сервери стануть доступними
	if err := waitForServersReady(); err != nil {
		t.Fatalf("Servers not ready: %v", err)
	}

	client := http.Client{Timeout: timeout}
	servers := make(map[string]int)

	for i := 0; i < requestsCount; i++ {
		resp, err := client.Get(lbAddr)
		if err != nil {
			t.Errorf("Request %d failed: %v", i+1, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d for request %d", resp.StatusCode, i+1)
		}

		serverID := resp.Header.Get("lb-from")
		if serverID == "" {
			t.Errorf("Missing 'lb-from' header in request %d", i+1)
		} else {
			servers[serverID]++
			t.Logf("Request %d handled by server: %s", i+1, serverID)
			// Додаємо затримку для імітації реального навантаження
			time.Sleep(50 * time.Millisecond)
		}
	}

	if len(servers) < 2 {
		t.Errorf("Least Connections algorithm should distribute requests to multiple servers. Got %d unique servers", len(servers))
	} else {
		t.Logf("Requests distributed to %d unique servers", len(servers))
		for server, count := range servers {
			t.Logf("Server %s handled %d requests", server, count)
		}
	}
}

// Чекаємо готовності всіх серверів
func waitForServersReady() error {
	servers := []string{"server1:8080", "server2:8080", "server3:8080"}
	client := http.Client{Timeout: retryInterval}

	for _, server := range servers {
		success := false
		for attempt := 0; attempt < maxRetries; attempt++ {
			resp, err := client.Get("http://" + server + healthEndpoint)
			if err == nil && resp.StatusCode == http.StatusOK {
				success = true
				break
			}
			time.Sleep(retryInterval)
		}
		if !success {
			return fmt.Errorf("server %s not ready after %d attempts", server, maxRetries)
		}
	}
	return nil
}
