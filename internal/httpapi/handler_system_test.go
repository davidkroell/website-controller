package httpapi

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
	"website-operator/httpapiclient"
	"website-operator/internal"

	"github.com/stretchr/testify/assert"
)

var apiClient *httpapiclient.Client

func TestMain(m *testing.M) {
	baseURL := internal.FromEnvWithDefault("TEST_API_BASEURL", "http://localhost:8082/")

	var err error
	apiClient, err = httpapiclient.NewDefaultClient(baseURL)
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestSystem_CreateListUpdateDeleteWebsite(t *testing.T) {
	sitename := genRandom("website")
	const hostname = "test.minikube.local"
	const nginx128 = "docker.io/nginx:1.28"
	const nginx129 = "docker.io/nginx:1.29"
	htmlContents := genRandom("html-body-content")

	_, err := apiClient.CreateWebsite(context.Background(), httpapiclient.WebsiteCreateDTO{
		WebsiteBase: httpapiclient.WebsiteBase{
			HtmlContent: htmlContents,
			Hostname:    hostname,
			NginxImage:  nginx128,
		},
		Name: sitename,
	})
	defer apiClient.DeleteWebsite(context.Background(), sitename) // make sure website is deleted always (in case of test fails)
	assert.NoError(t, err)

	resp, err := waitUntilHttpStatus(hostname, 30*time.Second, http.StatusOK)
	assert.NoError(t, err)
	assertResponseHasContent(t, err, resp, htmlContents)

	// LIST websites

	websites, err := apiClient.ListWebsites(context.Background())
	site := getWebsiteByName(websites, sitename)
	assert.Equal(t, htmlContents, site.HtmlContent)
	assert.Equal(t, hostname, site.Hostname)
	assert.Equal(t, nginx128, site.NginxImage)
	assert.Equal(t, int64(1), site.Generation)
	assert.WithinDuration(t, time.Now(), site.CreationTimestamp, time.Minute)

	// UPDATE website
	const newHostname = "a.minikube.local"
	updatedHtmlContent := htmlContents + "-updated-from-test"
	updatedWebsite, err := apiClient.UpdateWebsite(context.Background(), sitename, httpapiclient.WebsiteUpdateDTO{
		WebsiteBase: httpapiclient.WebsiteBase{
			HtmlContent: updatedHtmlContent,
			Hostname:    newHostname,
			NginxImage:  nginx129,
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, newHostname, updatedWebsite.Hostname)
	assert.Equal(t, nginx129, updatedWebsite.NginxImage)
	assert.Equal(t, updatedHtmlContent, updatedWebsite.HtmlContent)

	resp, err = waitUntilHttpStatus(newHostname, 30*time.Second, http.StatusOK)
	assert.NoError(t, err)
	assertResponseHasContent(t, err, resp, updatedHtmlContent)

	// DELETE website
	err = apiClient.DeleteWebsite(context.Background(), sitename)
	assert.NoError(t, err)

	_, err = waitUntilHttpStatus(newHostname, 30*time.Second, http.StatusNotFound)
	assert.NoError(t, err)
}
