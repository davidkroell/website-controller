package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	webv1 "website-operator/api/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	testEnv   *envtest.Environment
	k8sClient client.Client
	cfg       *rest.Config
	scheme    = runtime.NewScheme()
)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	// cancel() // TODO check
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func TestMain(m *testing.M) {
	RegisterFailHandler(Fail)
	// Start envtest
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "yaml")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Register schemes
	_ = webv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)

	// Create client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Run tests
	code := m.Run()

	// Teardown
	_ = testEnv.Stop()
	os.Exit(code)
}

func TestWebsiteCreatesResources(t *testing.T) {
	g := NewWithT(t)
	ctrl.SetLogger(zap.New())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// controller-runtime manager
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme})
	g.Expect(err).ToNot(HaveOccurred())

	// client-go clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	g.Expect(err).ToNot(HaveOccurred())

	// register controller
	reconciler := NewWebsiteController(k8sManager, clientset)
	err = ctrl.NewControllerManagedBy(k8sManager).
		For(&webv1.WebSite{}).
		Complete(reconciler)
	g.Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	// --- Create Website CR ---
	website := &webv1.WebSite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-site",
			Namespace: "default",
		},
		Spec: webv1.WebSiteSpec{
			HtmlContent: "test-html-content",
			Hostname:    "test.anexia.com",
			NginxImage:  "docker.io/nginx:1.28",
		},
	}
	err = k8sClient.Create(ctx, website)
	g.Expect(err).ToNot(HaveOccurred())

	deployList := &appsv1.DeploymentList{}
	err = k8sClient.List(ctx, deployList)
	g.Expect(err).ToNot(HaveOccurred())

	// --- Assert Deployment created ---
	deploy := &appsv1.Deployment{}
	g.Eventually(func() bool {
		err := k8sClient.Get(ctx,
			types.NamespacedName{Name: "website-test-site-deploy", Namespace: "default"},
			deploy)
		return err == nil
	}, 10*time.Second, 500*time.Millisecond).Should(BeTrue())

	// --- Assert Service created ---
	cm := &corev1.ConfigMap{}
	g.Eventually(func() bool {
		err := k8sClient.Get(ctx,
			types.NamespacedName{Name: "website-test-site-cm", Namespace: "default"},
			cm)
		return err == nil
	}, 10*time.Second, 500*time.Millisecond).Should(BeTrue())

	// --- Assert Service created ---
	svc := &corev1.Service{}
	g.Eventually(func() bool {
		err := k8sClient.Get(ctx,
			types.NamespacedName{Name: "website-test-site-service", Namespace: "default"},
			svc)
		return err == nil
	}, 10*time.Second, 500*time.Millisecond).Should(BeTrue())

	// --- Assert Ingress created ---
	ing := &networkingv1.Ingress{}
	g.Eventually(func() bool {
		err := k8sClient.Get(ctx,
			types.NamespacedName{Name: "website-test-site-ingress", Namespace: "default"},
			ing)
		return err == nil
	}, 10*time.Second, 500*time.Millisecond).Should(BeTrue())

}
