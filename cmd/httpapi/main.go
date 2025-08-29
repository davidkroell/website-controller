package main

import (
	webv1 "website-operator/api/v1"
	webv1client "website-operator/clientset/v1"
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

	runtime.Must(webv1.AddToScheme(scheme.Scheme))
	clientSet, err := webv1client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	handler := httpapi.NewWebsiteHandler(clientSet)

	router := httpapi.NewRouter(handler)

	addr := internal.FromEnvWithDefault("HTTPAPI_LISTEN_ADDR", ":8082")

	panic(router.Run(addr))
}
