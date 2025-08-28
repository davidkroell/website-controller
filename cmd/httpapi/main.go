package main

import (
	"net/http"
	"strings"
	"time"
	wsapiv1 "website-operator/api/v1"
	wsapiv1Client "website-operator/clientset/v1"
	"website-operator/internal"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func NewRouter(handler WebsiteHandler) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api")
	{
		api.GET("/websites", handler.List)
		api.POST("/websites", handler.Create)
		//api.PUT("/websites", handler.Update)
		//api.DELETE("/websites", handler.Delete)
	}

	return r
}

type WebsiteHandler struct {
	kubeClient wsapiv1Client.WebsiteV1Interface
}

func mapKubeWebsiteToDTO(site *wsapiv1.WebSite) *WebsiteDTO {
	return &WebsiteDTO{
		Name:              site.Name,
		HtmlContent:       site.Spec.HtmlContent,
		Hostname:          site.Spec.Hostname,
		NginxImage:        site.Spec.NginxImage,
		Labels:            site.Labels,
		Generation:        site.Generation,
		CreationTimestamp: site.CreationTimestamp.Time,
	}
}

func mapKubeWebsiteListToDTO(sites *wsapiv1.WebSiteList) WebsiteListDTO {
	result := make(WebsiteListDTO, 0, len(sites.Items))

	for _, site := range sites.Items {
		result = append(result, mapKubeWebsiteToDTO(&site))
	}
	return result
}

func (h *WebsiteHandler) List(c *gin.Context) {
	sites, err := h.kubeClient.Websites("default").List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := mapKubeWebsiteListToDTO(sites)

	c.JSON(http.StatusOK, result)
}

func (h *WebsiteHandler) Create(c *gin.Context) {
	var dto WebsiteCreateDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !strings.HasPrefix(dto.NginxImage, "docker.io/nginx:") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nginx image is invalid"})
		return
	}

	newSite, err := h.kubeClient.Websites("default").Create(c.Request.Context(), &wsapiv1.WebSite{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WebSite",
			APIVersion: "anexia.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: dto.Name,
		},
		Spec: wsapiv1.WebSiteSpec{
			HtmlContent: dto.HtmlContent,
			Hostname:    dto.Hostname,
			NginxImage:  dto.NginxImage,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, mapKubeWebsiteToDTO(newSite))
}

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

type WebsiteUpdateDTO struct {
	HtmlContent string `json:"htmlContent"`
	Hostname    string `json:"hostname"`
	NginxImage  string `json:"nginxImage"`
}

func main() {
	_, config, err := internal.GetLocalOrInClusterKubernetes()
	if err != nil {
		panic(err.Error())
	}

	runtime.Must(wsapiv1.AddToScheme(scheme.Scheme))
	clientSet, err := wsapiv1Client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	handler := WebsiteHandler{
		kubeClient: clientSet,
	}

	router := NewRouter(handler)
	panic(router.Run(":8080"))
}
