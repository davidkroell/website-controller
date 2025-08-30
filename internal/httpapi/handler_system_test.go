package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"
	"website-operator/httpapiclient"
	"website-operator/internal"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var apiClient *httpapiclient.Client

func TestHandlerSystemTestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HttpApi SystemTest Suite")
}

var _ = BeforeSuite(func() {
	By("setting up api client")

	baseURL := internal.FromEnvWithDefault("TEST_API_BASEURL", "http://localhost:8082/")

	var err error
	apiClient, err = httpapiclient.NewDefaultClient(baseURL)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("SystemTest HttpApi", Ordered, func() {
	sitename := genRandom("website")
	const hostname = "test.minikube.local"
	const newHostname = "a.minikube.local"

	const nginx128 = "docker.io/nginx:1.28"
	const nginx129 = "docker.io/nginx:1.29"
	htmlContents := genRandom("html-body-content")

	It("should create website", func() {
		_, err := apiClient.CreateWebsite(context.Background(), httpapiclient.WebsiteCreateDTO{
			WebsiteBase: httpapiclient.WebsiteBase{
				HtmlContent: htmlContents,
				Hostname:    hostname,
				NginxImage:  nginx128,
			},
			Name: sitename,
		})
		Expect(err).ToNot(HaveOccurred())

		resp, err := waitUntilHttpStatus(hostname, 30*time.Second, http.StatusOK)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp).To(HaveHTTPBody(htmlContents))
	})

	It("should be able to list websites", func() {
		websites, err := apiClient.ListWebsites(context.Background())
		Expect(err).ToNot(HaveOccurred())
		site := getWebsiteByName(websites, sitename)
		Expect(site.HtmlContent).To(Equal(htmlContents))
		Expect(site.Hostname).To(Equal(hostname))
		Expect(site.NginxImage).To(Equal(nginx128))
		Expect(site.Generation).To(BeEquivalentTo(1))
		Expect(site.CreationTimestamp).To(BeTemporally("~", time.Now(), 2*time.Minute))
	})

	It("should be able to update websites", func() {
		updatedHtmlContent := htmlContents + "-updated-from-test"
		updatedWebsite, err := apiClient.UpdateWebsite(context.Background(), sitename, httpapiclient.WebsiteUpdateDTO{
			WebsiteBase: httpapiclient.WebsiteBase{
				HtmlContent: updatedHtmlContent,
				Hostname:    newHostname,
				NginxImage:  nginx129,
			},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedWebsite.Hostname).To(Equal(newHostname))
		Expect(updatedWebsite.NginxImage).To(Equal(nginx129))
		Expect(updatedWebsite.HtmlContent).To(Equal(updatedHtmlContent))

		resp, err := waitUntilHttpStatus(newHostname, 30*time.Second, http.StatusOK)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp).To(HaveHTTPBody(updatedHtmlContent))
	})

	It("should be able to delete websites", func() {
		err := apiClient.DeleteWebsite(context.Background(), sitename)
		Expect(err).ToNot(HaveOccurred())

		_, err = waitUntilHttpStatus(newHostname, 30*time.Second, http.StatusNotFound)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterAll(func() {
		apiClient.DeleteWebsite(context.Background(), sitename)
	})
})
