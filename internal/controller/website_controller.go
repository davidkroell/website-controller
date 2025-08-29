package controller

import (
	"context"
	"fmt"
	webv1 "website-operator/api/v1"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type WebsiteController struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient kubernetes.Interface
}

func NewWebsiteController(mgr manager.Manager, kubeClient kubernetes.Interface) *WebsiteController {
	return &WebsiteController{
		Client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		kubeClient: kubeClient,
	}
}

func (r *WebsiteController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	website, err := r.getWebsite(ctx, req)
	if r.needsFinalizeWebsite(err) {
		return r.finalizeWebsite(ctx, req)
	}

	if err = r.ensureDeployment(ctx, req, website); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.ensureConfigMap(ctx, req, website); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.ensureService(ctx, req); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.ensureIngress(ctx, req, website); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *WebsiteController) siteName(req ctrl.Request) string {
	return "website-" + req.Name
}

func (r *WebsiteController) getWebsite(ctx context.Context, req ctrl.Request) (*webv1.WebSite, error) {
	var website webv1.WebSite
	err := r.Client.Get(ctx, req.NamespacedName, &website)
	return &website, err
}

func (r *WebsiteController) needsFinalizeWebsite(err error) bool {
	return err != nil && errors.IsNotFound(err)
}

func (r *WebsiteController) finalizeWebsite(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	siteName := r.siteName(req)
	log := log.FromContext(ctx)

	deploymentsClient := r.kubeClient.AppsV1().Deployments(req.Namespace)
	cmClient := r.kubeClient.CoreV1().ConfigMaps(req.Namespace)
	svcClient := r.kubeClient.CoreV1().Services(req.Namespace)
	ingressClient := r.kubeClient.NetworkingV1().Ingresses(req.Namespace)

	err := deploymentsClient.Delete(ctx, DeploymentObjectName(siteName), metav1.DeleteOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't finalized deployment: %s", err)
	}
	log.Info("finalized deployment for website")
	err = cmClient.Delete(ctx, ConfigMapObjectName(siteName), metav1.DeleteOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't finalized configmap: %s", err)
	}
	log.Info("finalized configmap for website")

	err = svcClient.Delete(ctx, ServiceObjectName(siteName), metav1.DeleteOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't finalized service: %s", err)
	}
	log.Info("finalized service for website")

	err = ingressClient.Delete(ctx, IngressObjectName(siteName), metav1.DeleteOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't finalized ingress: %s", err)
	}
	log.Info("finalized ingress for website")

	return ctrl.Result{}, err
}

func (r *WebsiteController) ensureDeployment(ctx context.Context, req ctrl.Request, website *webv1.WebSite) error {
	siteName := r.siteName(req)
	log := log.FromContext(ctx)
	deploymentsClient := r.kubeClient.AppsV1().Deployments(req.Namespace)

	deployment, err := deploymentsClient.Get(ctx, DeploymentObjectName(siteName), metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		deploymentObj := CreateDeploymentObject(siteName, website.Spec)
		_, err := deploymentsClient.Create(ctx, deploymentObj, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("couldn't create deployment: %s", err)
		}

		log.Info("new deployment created for website")
		return nil
	}

	// look up nginx image change
	if r.ensureDeploymentSpec(deployment, website) {
		deployment, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("couldn't update deployment: %s", err)
		}
		log.Info("updated deployment")
	}

	return nil
}

func (r *WebsiteController) ensureDeploymentSpec(deployment *v1.Deployment, website *webv1.WebSite) bool {
	needsUpdate := deployment.Spec.Template.Spec.Containers[0].Image != website.Spec.NginxImage

	deployment.Spec.Template.Spec.Containers[0].Image = website.Spec.NginxImage
	return needsUpdate
}

func (r *WebsiteController) ensureConfigMap(ctx context.Context, req ctrl.Request, website *webv1.WebSite) error {
	siteName := r.siteName(req)
	log := log.FromContext(ctx)

	cmClient := r.kubeClient.CoreV1().ConfigMaps(req.Namespace)

	// look up config map differences in HTML contents
	confMap, err := cmClient.Get(ctx, ConfigMapObjectName(siteName), metav1.GetOptions{})

	if err != nil && errors.IsNotFound(err) {
		cmObj := CreateConfigMapObject(siteName, website.Spec)
		_, err = cmClient.Create(ctx, cmObj, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("couldn't create configmap: %s", err)
		}
		log.Info("new configmap created for website")
		return nil
	}

	if r.ensureConfigMapSpec(confMap, website) {
		_, err = cmClient.Update(ctx, confMap, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("couldn't update ConfigMap: %s", err)
		}
		log.Info("website contents updated via configmap")
	}
	return nil
}

func (r *WebsiteController) ensureConfigMapSpec(confMap *corev1.ConfigMap, website *webv1.WebSite) bool {
	needsUpdate := confMap.Data["index.html"] != website.Spec.HtmlContent
	confMap.Data["index.html"] = website.Spec.HtmlContent
	return needsUpdate
}

func (r *WebsiteController) ensureService(ctx context.Context, req ctrl.Request) error {
	siteName := r.siteName(req)
	log := log.FromContext(ctx)

	svcClient := r.kubeClient.CoreV1().Services(req.Namespace)

	_, err := svcClient.Get(ctx, ServiceObjectName(siteName), metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		svcObject := CreateServiceObject(siteName)
		svcObject, err = svcClient.Create(ctx, svcObject, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("couldn't create service: %s", err)
		}

		log.Info("new service created for website")
		return nil
	}

	// service does not need update because website spec does not influence service

	return nil
}

func (r *WebsiteController) ensureIngress(ctx context.Context, req ctrl.Request, website *webv1.WebSite) error {
	siteName := r.siteName(req)
	log := log.FromContext(ctx)

	ingressClient := r.kubeClient.NetworkingV1().Ingresses(req.Namespace)

	ingress, err := ingressClient.Get(ctx, IngressObjectName(siteName), metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		ingressObject := CreateIngressObj(siteName, website.Spec)
		ingressObject, err = ingressClient.Create(ctx, ingressObject, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("couldn't create ingress: %s", err)
		}

		log.Info("new ingress created for website, exposed now via hostname", "hostname", website.Spec.Hostname)

		return nil
	}

	if r.ensureIngressSpec(ingress, website) {
		_, err = ingressClient.Update(ctx, ingress, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("couldn't update ingress spec: %s", err)
		}

		log.Info("ingress spec updated")
	}
	return nil
}

func (r *WebsiteController) ensureIngressSpec(ingress *netv1.Ingress, website *webv1.WebSite) bool {
	needsUpdate := ingress.Spec.Rules[0].Host != website.Spec.Hostname
	ingress.Spec.Rules[0].Host = website.Spec.Hostname
	return needsUpdate
}
