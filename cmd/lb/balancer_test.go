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

	//previously prepared urls with known hash
	hash0 := hash("http://localhost1")
	hash2 := hash("http://localhost2")
	hash1 := hash("http://localhost0")

	index0 := hash0 % len(healthyServersPool)
	index2 := hash2 % len(healthyServersPool)
	index1 := hash1 % len(healthyServersPool)
	c.Assert(index0, check.Equals, 0)
	c.Assert(index2, check.Equals, 2)
	c.Assert(index1, check.Equals, 1)
}
