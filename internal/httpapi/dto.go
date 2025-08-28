package httpapi

import (
	"fmt"
	"strings"
	"time"
)

type WebsiteListDTO []*WebsiteDTO

type WebsiteDTO struct {
	Name        string `json:"name"`
	HtmlContent string `json:"htmlContent"`
	Hostname    string `json:"hostname"`
	NginxImage  string `json:"nginxImage"`

	Labels            map[string]string `json:"labels"`
	Generation        int64             `json:"generation"`
	CreationTimestamp time.Time         `json:"creationTimestamp"`
}

type WebsiteCreateDTO struct {
	Name        string `json:"name"`
	HtmlContent string `json:"htmlContent"`
	Hostname    string `json:"hostname"`
	NginxImage  string `json:"nginxImage"`
}

func (w *WebsiteCreateDTO) Validate() error {
	// TODO deduplicate models and validation
	if !strings.HasPrefix(w.NginxImage, "docker.io/nginx:") {
		return fmt.Errorf("nginx image '%s' is invalid. must start with 'docker.io/nginx:'", w.NginxImage)
	}
	return nil
}

func (w *WebsiteUpdateDTO) Validate() error {
	// TODO deduplicate models and validation

	if !strings.HasPrefix(w.NginxImage, "docker.io/nginx:") {
		return fmt.Errorf("nginx image '%s' is invalid. must start with 'docker.io/nginx:'", w.NginxImage)
	}
	return nil
}

type WebsiteUpdateDTO struct {
	HtmlContent string `json:"htmlContent"`
	Hostname    string `json:"hostname"`
	NginxImage  string `json:"nginxImage"`
}
