package internal

import (
	"context"
	"fmt"
	wsapiv1 "website-operator/api/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type WebsiteReconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

func NewWebsiteReconciler(mgr manager.Manager, clientset *kubernetes.Clientset) *WebsiteReconciler {
	return &WebsiteReconciler{
		Client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		kubeClient: clientset,
	}
}

func (r *WebsiteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	siteName := "website-" + req.Name
	log := log.FromContext(ctx).WithValues("siteName", siteName)

	deploymentsClient := r.kubeClient.AppsV1().Deployments(req.Namespace)
	cmClient := r.kubeClient.CoreV1().ConfigMaps(req.Namespace)
	svcClient := r.kubeClient.CoreV1().Services(req.Namespace)
	ingressClient := r.kubeClient.NetworkingV1().Ingresses(req.Namespace)

	website, err := r.getWebsite(ctx, req)
	if r.needsFinalizeWebsite(err) {
		return r.finalizeWebsite(ctx, req, siteName)
	}

	deployment, err := deploymentsClient.Get(ctx, DeploymentObjectName(siteName), metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		// create new website object
		cmObj := CreateConfigMapObject(siteName, website.Spec)
		_, err = cmClient.Create(ctx, cmObj, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, fmt.Errorf("couldn't create configmap: %s", err)
		}

		deploymentObj := CreateDeploymentObject(siteName, website.Spec)
		_, err := deploymentsClient.Create(ctx, deploymentObj, metav1.CreateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't create deployment: %s", err)
		}

		log.Info("new deployment created for website")

		svcObject := CreateServiceObject(siteName)
		svcObject, err = svcClient.Create(ctx, svcObject, metav1.CreateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't create service: %s", err)
		}

		log.Info("new service created for website")

		ingressObject := CreateIngressObj(siteName, website.Spec)
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
	ingressSpec, err := ingressClient.Get(ctx, IngressObjectName(siteName), metav1.GetOptions{})
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
	confMap, err := cmClient.Get(ctx, ConfigMapObjectName(siteName), metav1.GetOptions{})
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

func (r *WebsiteReconciler) getWebsite(ctx context.Context, req ctrl.Request) (*wsapiv1.WebSite, error) {
	var website wsapiv1.WebSite
	err := r.Client.Get(ctx, req.NamespacedName, &website)
	return &website, err
}

func (r *WebsiteReconciler) needsFinalizeWebsite(err error) bool {
	return err != nil && errors.IsNotFound(err)
}

func (r *WebsiteReconciler) finalizeWebsite(ctx context.Context, req ctrl.Request, siteName string) (ctrl.Result, error) {
	deploymentsClient := r.kubeClient.AppsV1().Deployments(req.Namespace)
	cmClient := r.kubeClient.CoreV1().ConfigMaps(req.Namespace)
	svcClient := r.kubeClient.CoreV1().Services(req.Namespace)
	ingressClient := r.kubeClient.NetworkingV1().Ingresses(req.Namespace)

	err := deploymentsClient.Delete(ctx, DeploymentObjectName(siteName), metav1.DeleteOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't delete deployment: %s", err)
	}
	err = cmClient.Delete(ctx, ConfigMapObjectName(siteName), metav1.DeleteOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't delete configmap: %s", err)
	}
	err = svcClient.Delete(ctx, ServiceObjectName(siteName), metav1.DeleteOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't delete service: %s", err)
	}
	err = ingressClient.Delete(ctx, IngressObjectName(siteName), metav1.DeleteOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't delete ingress: %s", err)
	}
	return ctrl.Result{}, err
}
