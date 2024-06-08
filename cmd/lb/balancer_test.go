package main

import (
	"gopkg.in/check.v1"
	"io/ioutil"
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

	req0 := httptest.NewRequest("GET", "http://localhost1", nil)
	req1 := httptest.NewRequest("GET", "http://localhost0", nil)
	req2 := httptest.NewRequest("GET", "http://localhost2", nil)

	w := httptest.NewRecorder()
	forward(healthyServersPool[index0], w, req0)
	resp1 := w.Result()
	body1, _ := ioutil.ReadAll(resp1.Body)
	c.Assert(string(body1), check.Equals, "server1")

	w = httptest.NewRecorder()
	forward(healthyServersPool[index1], w, req1)
	resp0 := w.Result()
	body0, _ := ioutil.ReadAll(resp0.Body)
	c.Assert(string(body0), check.Equals, "server2")

	w = httptest.NewRecorder()
	forward(healthyServersPool[index2], w, req2)
	resp2 := w.Result()
	body2, _ := ioutil.ReadAll(resp2.Body)
	c.Assert(string(body2), check.Equals, "server3")
}
