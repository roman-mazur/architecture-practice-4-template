package integration

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = &http.Client{
	Timeout: 3 * time.Second,
}

func TestBalancer(t *testing.T) {
	if !isIntegrationTestEnabled() {
		t.Skip("Integration test is not enabled")
	}

	if !isBalancerAvailable() {
		t.Skip("Balancer is not available")
	}

	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
	if err != nil {
		t.Fatalf("Failed to get response from balancer: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Response from balancer: [%s]", resp.Header.Get("lb-from"))
}

func isIntegrationTestEnabled() bool {
	_, exists := os.LookupEnv("INTEGRATION_TEST")
	return exists
}

func isBalancerAvailable() bool {
	resp, err := client.Get(baseAddress)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func BenchmarkBalancer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			b.Errorf("Request failed: %v", err)
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
}
