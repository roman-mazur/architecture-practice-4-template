package main

import (
  "bytes"
  "io"
  "net/http"
  "net/http/httptest"
  "strings"
  "testing"
  "time"

  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/suite"
)

type FakeAlwaysTrueHealthChecker struct{}

func (hc *FakeAlwaysTrueHealthChecker) Check(server string) bool {
  return true
}

type FakeReturnRequestBodyRequestSender struct{}

func (rs *FakeReturnRequestBodyRequestSender) Send(request *http.Request) (*http.Response, error) {
  bodyBytes, err := io.ReadAll(request.Body)
  if err != nil {
    return nil, err
  }
  return &http.Response{
    StatusCode: 200,
    Proto:      "HTTP/1.1",
    Header:     make(http.Header),
    Body:       io.NopCloser(bytes.NewBuffer(bodyBytes)),
    Request:    &http.Request{},
    Close:      false,
  }, nil
}

type MySuite struct {
  suite.Suite
}

func TestMySuite(t *testing.T) {
  suite.Run(t, new(MySuite))
}

func (s *MySuite) TestScheme() {
  *https = true
  assert.Equal(s.T(), "https", scheme())

  *https = false
  assert.Equal(s.T(), "http", scheme())
}

func (s *MySuite) TestBalancer() {
  healthChecker = &FakeAlwaysTrueHealthChecker{}
  requestSender = &FakeReturnRequestBodyRequestSender{}
  serversPool = []string{"http://server1:1", "http://server2:1", "http://server3:1"}
  healthCheck(serversPool)
  time.Sleep(10 * time.Second)

  server := chooseServer()
  assert.NotNil(s.T(), server)
  assert.Contains(s.T(), server, "http://server")

  err := forward("http://server1:1", httptest.NewRecorder(), httptest.NewRequest("GET", "http://server1:1", strings.NewReader("body length 14")))
  assert.NoError(s.T(), err)
  err = forward("http://server3:1", httptest.NewRecorder(), httptest.NewRequest("GET", "http://server3:1", strings.NewReader("body length 14")))
  assert.NoError(s.T(), err)

  server = chooseServer()
  assert.Equal(s.T(), "http://server2:1", server)
}
