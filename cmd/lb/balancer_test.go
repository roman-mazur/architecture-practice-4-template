package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gopkg.in/check.v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type LoadBalancerSuite struct{}

var _ = check.Suite(&LoadBalancerSuite{})

func (s *LoadBalancerSuite) SetUpTest(c *check.C) {
	// This can be used to set up any necessary state before each test
	serversPool = []Server{
		{Address: "server1:8080", Traffic: 0},
		{Address: "server2:8080", Traffic: 0},
		{Address: "server3:8080", Traffic: 0},
	}
}

func (s *LoadBalancerSuite) TestSelectServer(c *check.C) {
	// Create a mock HTTP server
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "server1")
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "server2")
	}))
	defer server2.Close()

	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "server3")
	}))
	defer server3.Close()

	// Override serversPool with mock servers
	serversPool = []Server{
		{Address: server1.Listener.Addr().String(), Traffic: 100},
		{Address: server2.Listener.Addr().String(), Traffic: 50},
		{Address: server3.Listener.Addr().String(), Traffic: 0},
	}

	// Override the scheme function to return "http"
	scheme = func() string { return "http" }

	// Mock the timeout variable
	timeout = 1 * time.Second

	// Call selectServer and assert the results
	selectedServer, err := selectServer(serversPool)
	c.Assert(err, check.IsNil)
	c.Assert(selectedServer.Address, check.Equals, server2.Listener.Addr().String())
}

func (s *LoadBalancerSuite) TestSelectServerNoHealthyServers(c *check.C) {
	// Create a mock HTTP server
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "server1")
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "server2")
	}))
	defer server2.Close()

	// Override serversPool with mock servers
	serversPool = []Server{
		{Address: server1.Listener.Addr().String(), Traffic: 100},
		{Address: server2.Listener.Addr().String(), Traffic: 50},
	}

	// Override the scheme function to return "http"
	scheme = func() string { return "http" }

	// Mock the timeout variable
	timeout = 1 * time.Second

	// Call selectServer and assert the results
	selectedServer, err := selectServer(serversPool)
	c.Assert(err, check.NotNil)
	c.Assert(selectedServer, check.IsNil)
}
