package httpapi

import (
	"net/http"
	wsapiv1 "website-operator/api/v1"
	"website-operator/clientset/v1"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewRouter(handler WebsiteHandlerInterface) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api")
	{
		api.GET("/websites", handler.List)
		api.POST("/websites", handler.Create)
		api.PUT("/websites/:name", handler.Update)
		api.DELETE("/websites/:name", handler.Delete)
	}

	return r
}

type WebsiteHandler struct {
	kubeClient v1.WebsiteV1Interface
}

type WebsiteHandlerInterface interface {
	List(c *gin.Context)
	Create(c *gin.Context)
	Delete(c *gin.Context)
	Update(c *gin.Context)
}

func NewWebsiteHandler(kubeClient v1.WebsiteV1Interface) *WebsiteHandler {
	return &WebsiteHandler{
		kubeClient: kubeClient,
	}
}

func (h *WebsiteHandler) List(c *gin.Context) {
	sites, err := h.kubeClient.Websites("default").List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := MapKubeWebsiteListToDTO(sites)

	c.JSON(http.StatusOK, result)
}

func (h *WebsiteHandler) Create(c *gin.Context) {
	var dto WebsiteCreateDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dto.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	c.JSON(http.StatusCreated, MapKubeWebsiteToDTO(newSite))
}

func (h *WebsiteHandler) Delete(c *gin.Context) {
	err := h.kubeClient.Websites("default").Delete(c.Request.Context(), c.Param("name"), metav1.DeleteOptions{})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusAccepted)
}

func (h *WebsiteHandler) Update(c *gin.Context) {
	website, err := h.kubeClient.Websites("default").Get(c.Request.Context(), c.Param("name"), metav1.GetOptions{})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var dto WebsiteUpdateDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dto.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	website.Spec.HtmlContent = dto.HtmlContent
	website.Spec.Hostname = dto.Hostname
	website.Spec.NginxImage = dto.NginxImage

	site, err := h.kubeClient.Websites("default").Update(c.Request.Context(), website, metav1.UpdateOptions{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, MapKubeWebsiteToDTO(site))
}
