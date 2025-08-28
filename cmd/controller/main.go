package main

import (
	"context"
	"fmt"
	"os"
	wsapiv1 "website-operator/api/v1"
	"website-operator/internal"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		err = deploymentsClient.Delete(ctx, internal.DeploymentObjectName(siteName), metav1.DeleteOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't delete deployment: %s", err)
		}
		err = cmClient.Delete(ctx, internal.ConfigMapObjectName(siteName), metav1.DeleteOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't delete configmap: %s", err)
		}
		err = svcClient.Delete(ctx, internal.ServiceObjectName(siteName), metav1.DeleteOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't delete service: %s", err)
		}
		err = ingressClient.Delete(ctx, internal.IngressObjectName(siteName), metav1.DeleteOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't delete ingress: %s", err)
		}
		return ctrl.Result{}, err
	}

	deployment, err := deploymentsClient.Get(ctx, internal.DeploymentObjectName(siteName), metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		// create new website object
		cmObj := internal.CreateConfigMapObject(siteName, website.Spec)
		_, err = cmClient.Create(ctx, cmObj, metav1.CreateOptions{})
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			return ctrl.Result{}, fmt.Errorf("couldn't create configmap: %s", err)
		}

		deploymentObj := internal.CreateDeploymentObject(siteName, website.Spec)
		_, err := deploymentsClient.Create(ctx, deploymentObj, metav1.CreateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't create deployment: %s", err)
		}

		log.Info("new deployment created for website")

		svcObject := internal.CreateServiceObject(siteName)
		svcObject, err = svcClient.Create(ctx, svcObject, metav1.CreateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't create service: %s", err)
		}

		log.Info("new service created for website")

		ingressObject := internal.CreateIngressObj(siteName, website.Spec)
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
	ingressSpec, err := ingressClient.Get(ctx, internal.IngressObjectName(siteName), metav1.GetOptions{})
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
	confMap, err := cmClient.Get(ctx, internal.ConfigMapObjectName(siteName), metav1.GetOptions{})
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
