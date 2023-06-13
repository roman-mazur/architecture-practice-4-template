package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(BalancerSuite))
}

type BalancerSuite struct {
	suite.Suite
}

func (s *BalancerSuite) TestBalancer() {
	healthChecker := &HealthChecker{}
	healthChecker.healthyServers = []string{"server1:8080", "server2:8080", "server3:8080"}

	balancer := &Balancer{}
	balancer.healthChecker = healthChecker

	index1 := balancer.getServerIndexWithLowestLoad(map[string]int64{
		"server1:8080": 100,
		"server2:8080": 200,
		"server3:8080": 150,
	}, []string{"server1:8080", "server2:8080", "server3:8080"})

	index2 := balancer.getServerIndexWithLowestLoad(map[string]int64{
		"server1:8080": 300,
		"server2:8080": 200,
		"server3:8080": 250,
	}, []string{"server1:8080", "server2:8080", "server3:8080"})

	index3 := balancer.getServerIndexWithLowestLoad(map[string]int64{
		"server1:8080": 200,
		"server2:8080": 150,
		"server3:8080": 100,
	}, []string{"server1:8080", "server2:8080", "server3:8080"})

	assert.Equal(s.T(), 0, index1)
	assert.Equal(s.T(), 1, index2)
	assert.Equal(s.T(), 2, index3)
}

func (s *BalancerSuite) TestHealthChecker() {
	healthChecker := &HealthChecker{}
	healthChecker.health = func(s string) bool {
		if s == "1" {
			return false
		} else {
			return true
		}
	}

	healthChecker.serversPool = []string{"1", "2", "3"}
	healthChecker.healthyServers = []string{"4", "5", "6"}
	healthChecker.checkInterval = 1 * time.Second

	healthChecker.StartHealthCheck()

	time.Sleep(2 * time.Second)

	assert.Equal(s.T(), "2", healthChecker.GetHealthyServers()[0])
	assert.Equal(s.T(), "3", healthChecker.GetHealthyServers()[1])
	assert.Equal(s.T(), 2, len(healthChecker.GetHealthyServers()))
}
