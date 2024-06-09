package integration

import (
	"fmt"
	"gopkg.in/check.v1"
	"net/http"
	"os"
	"testing"
	"time"
)

type BalancerSuite struct{}

var _ = check.Suite(&BalancerSuite{})

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	// TODO: Реалізуйте інтеграційний тест для балансувальникка.
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
	if err != nil {
		t.Error(err)
	}
	t.Logf("response from [%s]", resp.Header.Get("lb-from"))
}

func (s *BalancerSuite) BenchmarkBalancer(c *check.C) {
	for i := 0; i < c.N; i++ {
		_, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			c.Error(err)
		}
	}
}
