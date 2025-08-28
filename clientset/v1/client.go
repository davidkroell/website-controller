package v1

import (
	"context"
	wsapiv1 "website-operator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type WebsiteV1Interface interface {
	Websites(namespace string) WebsiteInterface
}

type WebsiteV1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*WebsiteV1Client, error) {
	config := *c
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: wsapiv1.GroupName, Version: wsapiv1.GroupVersion}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &WebsiteV1Client{restClient: client}, nil
}

func (c *WebsiteV1Client) Websites(namespace string) WebsiteInterface {
	return &websiteClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

type WebsiteInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*wsapiv1.WebSiteList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*wsapiv1.WebSite, error)
	Create(ctx context.Context, site *wsapiv1.WebSite, opts metav1.CreateOptions) (*wsapiv1.WebSite, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type websiteClient struct {
	restClient rest.Interface
	ns         string
}

func (c *websiteClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*wsapiv1.WebSite, error) {
	result := wsapiv1.WebSite{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("websites").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *websiteClient) Create(ctx context.Context, website *wsapiv1.WebSite, opts metav1.CreateOptions) (*wsapiv1.WebSite, error) {
	result := wsapiv1.WebSite{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource("websites").
		Body(website).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *websiteClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	err := c.restClient.
		Delete().
		Namespace(c.ns).
		Resource("websites").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()

	return err
}

func (c *websiteClient) List(ctx context.Context, opts metav1.ListOptions) (*wsapiv1.WebSiteList, error) {
	result := wsapiv1.WebSiteList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("websites").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}
