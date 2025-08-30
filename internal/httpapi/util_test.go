package httpapi

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"
	"website-operator/httpapiclient"
)

func genRandom(prefix string) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

	const n = 8

	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to read random bytes: %v", err))
	}
	for i := 0; i < n; i++ {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return "ci-" + prefix + "-" + string(b)
}

func waitUntilHttpStatus(url string, d time.Duration, status int) (*http.Response, error) {
	return waitUntilHttpFulfills(url, d, func(resp *http.Response) bool {
		return resp.StatusCode == status
	})
}

func waitUntilHttpFulfills(url string, d time.Duration, predicate func(resp *http.Response) bool) (*http.Response, error) {
	if !strings.HasPrefix(url, "http://") || !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	backoff := 500 * time.Millisecond
	deadline := time.Now().Add(d)

	for {
		// Try request
		resp, err := http.Get(url)
		if err == nil && predicate(resp) {
			return resp, nil // fulfills predicate
		}

		// Check timeout
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("service at %s not available within %v", url, d)
		}

		// Sleep with exponential backoff, capped at 5s
		time.Sleep(backoff)
		backoff *= 2
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
	}
}

func getWebsiteByName(s httpapiclient.WebsiteListDTO, name string) *httpapiclient.WebsiteDTO {
	for _, w := range s {
		if w.Name == name {
			return w
		}
	}
	return nil
}
