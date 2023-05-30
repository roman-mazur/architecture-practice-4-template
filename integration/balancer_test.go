package integration

import (
	"fmt"
	. "gopkg.in/check.v1"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

type IntegrationTestSuite struct{}

var _ = Suite(&IntegrationTestSuite{})

func TestBalancer(t *testing.T) {

	TestingT(t)

}

func (s *IntegrationTestSuite) TestGetRequest(c *C) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		c.Skip("Integration test is not enabled")
	}

	for i := 0; i < 3; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			c.Error(err)
		}
		c.Check(resp.Header.Get("lb-from"), Equals, "server"+string(rune(i+1))+":8080")
	}

}

func BenchmarkBalancer(b *testing.B) {
	// TODO: Реалізуйте інтеграційний бенчмарк для балансувальникка.
}
