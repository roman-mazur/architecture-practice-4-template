package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// echo «ok» для мок-бекендів
func ok(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }

func TestLeastConnections(t *testing.T) {
	/* --- 1. готуємо три фейкові бекенди -------------------------------- */
	s1 := httptest.NewServer(http.HandlerFunc(ok))
	s2 := httptest.NewServer(http.HandlerFunc(ok))
	s3 := httptest.NewServer(http.HandlerFunc(ok))
	defer s1.Close()
	defer s2.Close()
	defer s3.Close()

	// перезбираємо глобальний пул backends під тести
	backends = []*backend{
		{addr: s1.Listener.Addr().String()},
		{addr: s2.Listener.Addr().String()},
		{addr: s3.Listener.Addr().String()},
	}
	for _, be := range backends {
		be.healthy.Store(true)
	}

	/* --- 2. створюємо тестовий LB-сервер -------------------------------- */
	lb := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		be := pickBackend()
		require.NotNil(t, be, "must have healthy backend")
		forward(be, rw, r)
	}))
	defer lb.Close()

	/* --- 3. Пускаємо  ninety паралельних запитів ------------------------ */
	const N = 90
	var wg sync.WaitGroup
	wg.Add(N)

	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			resp, err := http.Get(lb.URL + "/")
			require.NoError(t, err)
			resp.Body.Close()
		}()
	}
	wg.Wait()

	/* --- 4. Перевіряємо, що розподіл ~рівномірний ---------------------- */
	require.Len(t, backends, 3)
	c0 := backends[0].connCnt.Load()
	c1 := backends[1].connCnt.Load()
	c2 := backends[2].connCnt.Load()
	require.InDelta(t, c0, c1, 2)
	require.InDelta(t, c1, c2, 2)
}
