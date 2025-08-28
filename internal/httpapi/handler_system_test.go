package httpapi

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
	"website-operator/httpapiclient"
	"website-operator/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var apiBinaryPath = internal.FromEnvWithDefault("TEST_API_BINARY_PATH", "../../bin/httpapi")

func RunSystemTestStandaloneAPI(t *testing.T, testFunc func(client *httpapiclient.Client, t *testing.T)) {
	baseURL := internal.FromEnvWithDefault("TEST_API_BASEURL", "http://localhost:8082/")

	client, err := httpapiclient.NewDefaultClient(baseURL)
	require.NoError(t, err)

	// run testFunc
	testFunc(client, t)
}

func RunSystemTest(t *testing.T, testFunc func(client *httpapiclient.Client, t *testing.T)) {
	// let the OS choose a port at random
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	baseURL := fmt.Sprintf("http://localhost:%d/", port)
	client, err := httpapiclient.NewDefaultClient(baseURL)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, apiBinaryPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("HTTPAPI_LISTEN_ADDR=:%d", port))

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	// Goroutine to stream stdout
	go streamOutput(ctx, t, stdout, "T_STDOUT")
	// Goroutine to stream stderr
	go streamOutput(ctx, t, stderr, "T_STDERR")

	// run testFunc
	testFunc(client, t)

	// Kill the process when test is done
	err = cmd.Process.Kill()
	require.NoError(t, err)
	_ = cmd.Wait()
}

// helper: copy lines to t.Log
func streamOutput(ctx context.Context, t *testing.T, r io.ReadCloser, name string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			t.Logf("[%s] %s", name, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		t.Logf("error reading %s: %v", name, err)
	}
}

func TestSystem_All(t *testing.T) {
	RunSystemTestStandaloneAPI(t, func(client *httpapiclient.Client, t *testing.T) {
		sitename := genRandom("website")
		const hostname = "test.minikube.local"
		const url = "http://" + hostname
		htmlContents := genRandom("html-body-content")

		_, err := client.CreateWebsite(context.Background(), httpapiclient.WebsiteCreateDTO{
			WebsiteBase: httpapiclient.WebsiteBase{
				HtmlContent: htmlContents,
				Hostname:    hostname,
				NginxImage:  "docker.io/nginx:1.28",
			},
			Name: sitename,
		})
		assert.NoError(t, err)

		assert.NoError(t, WaitUntilAvailable(url, 30*time.Second))

		resp, err := http.Get(url)
		assert.NoError(t, err)
		b, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		contents := string(b)
		assert.Equal(t, htmlContents, contents)
	})
}

func WaitUntilAvailable(url string, d time.Duration) error {
	backoff := 500 * time.Millisecond
	deadline := time.Now().Add(d)

	for {
		// Try request
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return nil // success
		}
		if resp != nil {
			_ = resp.Body.Close()
		}

		// Check timeout
		if time.Now().After(deadline) {
			return fmt.Errorf("service at %s not available within %v", url, d)
		}

		// Sleep with exponential backoff, capped at 5s
		time.Sleep(backoff)
		backoff *= 2
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
	}
}

const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

func genRandom(prefix string) string {
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
