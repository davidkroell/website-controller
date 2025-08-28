package httpapi

import "website-operator/api/v1"

func MapKubeWebsiteToDTO(site *v1.WebSite) *WebsiteDTO {
	return &WebsiteDTO{
		WebsiteBase: WebsiteBase{
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

func MapKubeWebsiteListToDTO(sites *v1.WebSiteList) WebsiteListDTO {
	result := make(WebsiteListDTO, 0, len(sites.Items))

	for _, site := range sites.Items {
		result = append(result, MapKubeWebsiteToDTO(&site))
	}
	return result
}
