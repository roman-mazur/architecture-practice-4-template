package main

import (
	"gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"testing"
)

type BalancerSuite struct{}

var _ = check.Suite(&BalancerSuite{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *BalancerSuite) TestBalancerComponents(c *check.C) {

	server1 := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("server1"))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("server2"))
	}))
	defer server2.Close()

	server3 := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("server3"))
	}))
	defer server3.Close()

	serversPool = []string{
		server1.Listener.Addr().String(),
		server2.Listener.Addr().String(),
		server3.Listener.Addr().String(),
	}

	healthyServersPool := healthyServers(serversPool)
	c.Assert(len(healthyServersPool), check.Equals, 3)

}
