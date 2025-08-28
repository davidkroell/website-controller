package httpapi

import (
	"website-operator/api/v1"
	"website-operator/httpapiclient"
)

func MapKubeWebsiteToDTO(site *v1.WebSite) *httpapiclient.WebsiteDTO {
	return &httpapiclient.WebsiteDTO{
		WebsiteBase: httpapiclient.WebsiteBase{
			HtmlContent: site.Spec.HtmlContent,
			Hostname:    site.Spec.Hostname,
			NginxImage:  site.Spec.NginxImage,
		},
		Name:              site.Name,
		Labels:            site.Labels,
		Generation:        site.Generation,
		CreationTimestamp: site.CreationTimestamp.Time,
	}
}

func MapKubeWebsiteListToDTO(sites *v1.WebSiteList) httpapiclient.WebsiteListDTO {
	result := make(httpapiclient.WebsiteListDTO, 0, len(sites.Items))

	for _, site := range sites.Items {
		result = append(result, MapKubeWebsiteToDTO(&site))
	}
	return result
}
