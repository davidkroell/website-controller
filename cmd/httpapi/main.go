package main

import (
	wsapiv1 "website-operator/api/v1"
	wsapiv1Client "website-operator/clientset/v1"
	"website-operator/internal"
	"website-operator/internal/httpapi"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

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

	handler := httpapi.NewWebsiteHandler(clientSet)

	router := httpapi.NewRouter(handler)
	panic(router.Run(":8082"))
}
