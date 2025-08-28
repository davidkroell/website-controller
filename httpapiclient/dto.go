package httpapiclient

import (
	"fmt"
	"strings"
	"time"
)

// WebsiteListDTO represents a list of websites.
type WebsiteListDTO []*WebsiteDTO

// WebsiteBase contains the common fields for all Website DTOs.
type WebsiteBase struct {
	HtmlContent string `json:"htmlContent"`
	Hostname    string `json:"hostname"`
	NginxImage  string `json:"nginxImage"`
}

// WebsiteDTO is the full website model returned by the API.
type WebsiteDTO struct {
	WebsiteBase

	Name              string            `json:"name"`
	Labels            map[string]string `json:"labels"`
	Generation        int64             `json:"generation"`
	CreationTimestamp time.Time         `json:"creationTimestamp"`
}

// WebsiteCreateDTO is used to create a new website.
type WebsiteCreateDTO struct {
	WebsiteBase
	Name string `json:"name"`
}

// WebsiteUpdateDTO is used to update an existing website.
type WebsiteUpdateDTO struct {
	WebsiteBase
}

func (w *WebsiteBase) Validate() error {
	if !strings.HasPrefix(w.NginxImage, "docker.io/nginx:") {
		return fmt.Errorf("nginx image '%s' is invalid. must start with 'docker.io/nginx:'", w.NginxImage)
	}
	return nil
}
