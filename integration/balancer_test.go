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

func (s *BalancerSuite) TestBalancer(c *check.C) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		c.Skip("Integration test is not enabled")
	}
	// TODO: Реалізуйте інтеграційний тест для балансувальникка.
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
	if err != nil {
		c.Error(err)
	}
	c.Logf("response from [%s]", resp.Header.Get("lb-from"))

	for i := 0; i < 5; i++ {
		resp0, err := client.Get(fmt.Sprintf("%s/zero", baseAddress))
		if err != nil {
			c.Error(err)
		}
		c.Check(resp0.Header.Get("lb-from"), check.Equals, "server1:8080")
	}

	for i := 0; i < 5; i++ {
		resp1, err := client.Get(fmt.Sprintf("%s/must_be_1", baseAddress))
		if err != nil {
			c.Error(err)
		}
		c.Check(resp1.Header.Get("lb-from"), check.Equals, "server2:8080")
	}

	for i := 0; i < 5; i++ {
		resp2, err := client.Get(fmt.Sprintf("%s/be_2", baseAddress))
		if err != nil {
			c.Error(err)
		}
		c.Check(resp2.Header.Get("lb-from"), check.Equals, "server3:8080")
	}
}

func (s *BalancerSuite) BenchmarkBalancer(c *check.C) {
	for i := 0; i < c.N; i++ {
		_, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			c.Error(err)
		}
	}
}

func Test(t *testing.T) {
	check.TestingT(t)
}
