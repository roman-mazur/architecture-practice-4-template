package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	assert := assert.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	assert.True(health(server.URL[len("http://"):]), "Health function should return true")
}

func TestForward(t *testing.T) {
	assert := assert.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", "http://localhost:8090", nil)
	assert.NoError(err)

	rw := httptest.NewRecorder()

	err = forward(server.URL[len("http://"):], rw, req)
	assert.NoError(err, "Forward function should not return error")
	assert.Equal(http.StatusOK, rw.Result().StatusCode, "Expected status code to be %v, but got %v", http.StatusOK, rw.Result().StatusCode)
}
