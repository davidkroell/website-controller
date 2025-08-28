package main

import (
	"context"
	"fmt"
	"os"
	wsapiv1 "website-operator/api/v1"
	"website-operator/internal"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup website controller")
)

func init() {
	utilruntime.Must(wsapiv1.AddToScheme(scheme))
}

type reconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

const (
	nginxPort = 80
)

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	siteName := "website-" + req.Name
	log := log.FromContext(ctx).WithValues("siteName", siteName)

	deploymentsClient := r.kubeClient.AppsV1().Deployments(req.Namespace)
	cmClient := r.kubeClient.CoreV1().ConfigMaps(req.Namespace)
	svcClient := r.kubeClient.CoreV1().Services(req.Namespace)
	ingressClient := r.kubeClient.NetworkingV1().Ingresses(req.Namespace)

	var website wsapiv1.WebSite
	err := r.Client.Get(ctx, req.NamespacedName, &website)
	if err != nil && k8serrors.IsNotFound(err) {
		// website not found -> delete it
		err = deploymentsClient.Delete(ctx, siteName, metav1.DeleteOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't delete deployment: %s", err)
		}
		err = cmClient.Delete(ctx, siteName, metav1.DeleteOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't delete configmap: %s", err)
		}
		err = svcClient.Delete(ctx, siteName, metav1.DeleteOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't delete service: %s", err)
		}
		err = ingressClient.Delete(ctx, siteName, metav1.DeleteOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't delete ingress: %s", err)
		}
		return ctrl.Result{}, err
	}

	deployment, err := deploymentsClient.Get(ctx, siteName, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		// create new website object
		cmObj := getConfigMapObject(siteName, website.Spec.HtmlContent)
		_, err = cmClient.Create(ctx, cmObj, metav1.CreateOptions{})
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			return ctrl.Result{}, fmt.Errorf("couldn't create configmap: %s", err)
		}

		deploymentObj := getDeploymentObject(siteName, website.Spec)
		_, err := deploymentsClient.Create(ctx, deploymentObj, metav1.CreateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't create deployment: %s", err)
		}

		log.Info("new website created")

		svcObject := getServiceObject(siteName)
		svcObject, err = svcClient.Create(ctx, svcObject, metav1.CreateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't create service: %s", err)
		}

		log.Info("new service created for website, exposed now via nodePort")

		ingressObject := getIngressObj(siteName, website.Spec.Hostname)
		ingressObject, err = ingressClient.Create(ctx, ingressObject, metav1.CreateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't create ingress: %s", err)
		}

		log.Info("new ingress created for website, exposed now via hostname", "hostname", website.Spec.Hostname)

		return ctrl.Result{}, nil
	}

	// look up nginx image change
	if deployment.Spec.Template.Spec.Containers[0].Image != website.Spec.NginxImage {
		// update deployment
		deployment.Spec.Template.Spec.Containers[0].Image = website.Spec.NginxImage

		deployment, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't update deployment: %s", err)
		}

		log.Info("updated deployment")
	}

	// look up hostname change
	ingressSpec, err := ingressClient.Get(ctx, siteName, metav1.GetOptions{})
	if err == nil {
		if ingressSpec.Spec.Rules[0].Host != website.Spec.Hostname {
			log.Info("ingress spec hostname need update")
			ingressSpec.Spec.Rules[0].Host = website.Spec.Hostname

			_, err = ingressClient.Update(ctx, ingressSpec, metav1.UpdateOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't update ingress spec: %s", err)
			}

			log.Info("ingress spec updated")
		}
	}

	// look up config map differences in HTML contents
	confMap, err := cmClient.Get(ctx, siteName, metav1.GetOptions{})
	if err == nil &&
		confMap.Data["index.html"] != website.Spec.HtmlContent {
		// config map differs, update required

		confMap.Data["index.html"] = website.Spec.HtmlContent

		_, err = cmClient.Update(ctx, confMap, metav1.UpdateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't update ConfigMap: %s", err)
		}
		log.Info("website contents updated via configmap")
	}

	log.Info("website is up-to-date")

	return ctrl.Result{}, nil
}

func ptr[T any](v T) *T {
	return &v
}

func getIngressObj(name, hostname string) *networkingv1.Ingress {
	const ingressClassName = "nginx"

	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ptr(ingressClassName),
			Rules: []networkingv1.IngressRule{
				{
					Host: hostname,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: ptr(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: name,
											Port: networkingv1.ServiceBackendPort{
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

func getServiceObject(name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"apptype":           "website",
				"anexia.com/expose": name,
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

func getDeploymentObject(name string, spec wsapiv1.WebSiteSpec) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr(int32(1)), // TODO replicas
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"apptype": "website",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"apptype":           "website",
						"anexia.com/expose": name,
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
										Name: name,
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

func getConfigMapObject(name, contents string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string]string{
			"index.html": contents,
		},
	}
}

func main() {
	clientset, config, err := internal.GetLocalOrInClusterKubernetes()
	if err != nil {
		panic(err.Error())
	}

	ctrl.SetLogger(zap.New())

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&wsapiv1.WebSite{}).
		Complete(&reconciler{
			Client:     mgr.GetClient(),
			scheme:     mgr.GetScheme(),
			kubeClient: clientset,
		})
	if err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "error running manager")
		os.Exit(1)
	}
}
