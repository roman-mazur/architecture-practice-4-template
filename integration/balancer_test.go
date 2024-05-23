package integration

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	var hosts []string

	// TODO: Реалізуйте інтеграційний тест для балансувальникка.
	for i := 0; i < 10; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()

		host := resp.Header.Get("lb-from");
		if (!contains(hosts, host)) {
			hosts = append(hosts, host)
		}
		t.Logf("response from [%s]", resp.Header.Get("lb-from"))
	}

	if len(hosts) < 3 {
		t.Errorf("expected at least 3 hosts, got %d", len(hosts))
	}
}

func BenchmarkBalancer(b *testing.B) {
    if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
        b.Skip("Integration test is not enabled")
    }

    minRequestsPerSecond := 5.0 // adjust as needed

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        totalRequests := 0
        for pb.Next() {
            resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
            if err != nil {
                b.Error(err)
            }
            resp.Body.Close()
            totalRequests++
        }
        b.Logf("Total requests: %d", totalRequests)
    })

	b.StopTimer()
    actualRequestsPerSecond := float64(b.N) / b.Elapsed().Seconds()
    b.Logf("Requests per second: %.2f", actualRequestsPerSecond)

    if actualRequestsPerSecond < minRequestsPerSecond {
        b.Errorf("Expected at least %.2f requests/second, but got %.2f", minRequestsPerSecond, actualRequestsPerSecond)
    }
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
