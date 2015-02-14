package httpservecache

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCacher(t *testing.T) {
	c := New("test", nil, 64)

	msg := []byte("Hello from expensive request\n")

	nGet := 0
	h := c.F(func(w http.ResponseWriter, r *http.Request) {
		nGet++
		t.Log("!!! Doing expensive request !!!")
		w.Write(msg)
	})

	r, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	const nReq = 10
	for i := 0; i < nReq; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("w.Code != http.StatusCode (=%d)", w.Code)
		}

		if string(w.Body.Bytes()) != string(msg) {
			t.Errorf("Response body != msg (=%q) ", msg)
		}
	}

	if nGet != 1 {
		t.Error("nGet != 1 (=%d)", nGet)
	}

	hits := c.Stats.CacheHits.Get()
	if hits != nReq-1 {
		t.Errorf("Cachehits: %d != %d", hits, nReq-1)
	}
	t.Logf("Cache stats: %#+v", c.Stats)
}
