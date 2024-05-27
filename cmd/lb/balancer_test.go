package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBalancer(t *testing.T) {
	// Створення мок серверів
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Server1"))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Server2"))
	}))
	defer server2.Close()

	// Оновлення пулу здорових серверів
	healthyPool = []string{server1.URL[len("http://"):], server2.URL[len("http://"):]}

	req := httptest.NewRequest("GET", "http://test.com/some-path", nil)
	w := httptest.NewRecorder()

	serverIndex := hashPath(req.URL.Path) % len(healthyPool)
	err := forward(healthyPool[serverIndex], w, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK, got %v", resp.StatusCode)
	}

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	resp.Body.Close()

	expectedBody := "Server1"
	if serverIndex == 1 {
		expectedBody = "Server2"
	}

	if string(body) != expectedBody {
		t.Fatalf("Expected body to be '%s', got %s", expectedBody, string(body))
	}
}
