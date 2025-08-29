package controller

import (
	webv1 "website-operator/api/v1"
	"website-operator/internal"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	nginxPort = 80

	ingressPath      = "/"
	ingressClassName = "nginx"

	websiteReplica = 1
)

func IngressObjectName(siteName string) string {
	return siteName + "-ingress"
}

func ServiceObjectName(siteName string) string {
	return siteName + "-service"
}

func DeploymentObjectName(siteName string) string {
	return siteName + "-deploy"
}

func ConfigMapObjectName(siteName string) string {
	return siteName + "-cm"
}

func CreateIngressObj(name string, spec webv1.WebSiteSpec) *netv1.Ingress {
	return &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: IngressObjectName(name),
		},
		Spec: netv1.IngressSpec{
			IngressClassName: internal.Ptr(ingressClassName),
			Rules: []netv1.IngressRule{
				{
					Host: spec.Hostname,
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     ingressPath,
									PathType: internal.Ptr(netv1.PathTypePrefix),
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: ServiceObjectName(name),
											Port: netv1.ServiceBackendPort{
												Number: nginxPort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func CreateServiceObject(name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: ServiceObjectName(name),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"apptype":           "website",
				"anexia.com/expose": DeploymentObjectName(name),
			},
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: "TCP",
					Port:     nginxPort,
					TargetPort: intstr.IntOrString{
						IntVal: nginxPort,
					},
				},
			},
		},
	}
}

func CreateDeploymentObject(name string, spec webv1.WebSiteSpec) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: DeploymentObjectName(name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: internal.Ptr(int32(websiteReplica)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"apptype": "website",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"apptype":           "website",
						"anexia.com/expose": DeploymentObjectName(name),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "website",
							Image: spec.NginxImage,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http-svc-port",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: nginxPort,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "contents",
									MountPath: "/usr/share/nginx/html",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "contents",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: ConfigMapObjectName(name),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func CreateConfigMapObject(name string, spec webv1.WebSiteSpec) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ConfigMapObjectName(name),
		},
		Data: map[string]string{
			"index.html": spec.HtmlContent,
		},
	}
}
