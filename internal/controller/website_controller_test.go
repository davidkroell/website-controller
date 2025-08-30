package controller

import (
	"context"
	"path/filepath"
	"testing"
	"time"
	webv1 "website-operator/api/v1"
	"website-operator/internal"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestWebsiteControllerSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "WebsiteController Suite")
}

var _ = BeforeSuite(func() {
	ctrl.SetLogger(zap.New())

	By("bootstrapping test environment")

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
	Expect(webv1.AddToScheme(scheme)).To(Succeed())
	Expect(appsv1.AddToScheme(scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(networkingv1.AddToScheme(scheme)).To(Succeed())

	// Create client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Create manager and controller
	ctx, cancel = context.WithCancel(context.Background())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme})
	Expect(err).ToNot(HaveOccurred())

	// client-go clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())

	// register controller
	reconciler := NewWebsiteController(k8sManager, clientset)
	err = ctrl.NewControllerManagedBy(k8sManager).
		For(&webv1.WebSite{}).
		Complete(reconciler)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	if cancel != nil {
		cancel()
	}
	Expect(testEnv.Stop()).To(Succeed())
})

var _ = Describe("WebsiteController", func() {
	It("should create a deployment, configmap, service and ingress from a website spec", func() {
		By("creating a website CR")
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
		err := k8sClient.Create(ctx, website)
		Expect(err).ToNot(HaveOccurred())

		By("wait for deployment to be created")
		deploy := &appsv1.Deployment{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx,
				types.NamespacedName{Name: "website-test-site-deploy", Namespace: "default"},
				deploy)
			return err == nil
		}, 10*time.Second, 500*time.Millisecond).Should(BeTrue())
		By("check deployment")
		Expect(deploy.Spec.Replicas).To(PointTo(BeEquivalentTo(1)))
		Expect(deploy.CreationTimestamp.Time).To(BeTemporally("~", time.Now(), 2*time.Minute))
		Expect(deploy.Spec.Selector.MatchLabels).To(HaveKeyWithValue("apptype", "website"))
		Expect(deploy.Spec.Template.ObjectMeta.Labels).To(
			And(HaveKeyWithValue("apptype", "website"), HaveKeyWithValue("anexia.com/expose", "website-test-site-deploy")))
		Expect(deploy.Spec.Template.Spec.Containers).To(HaveLen(1))
		Expect(deploy.Spec.Template.Spec.Containers[0].Name).To(Equal("website"))
		Expect(deploy.Spec.Template.Spec.Containers[0].Image).To(Equal("docker.io/nginx:1.28"))
		Expect(deploy.Spec.Template.Spec.Containers[0].VolumeMounts).To(Equal([]corev1.VolumeMount{
			{
				Name:      "contents",
				MountPath: "/usr/share/nginx/html",
			},
		}))

		Expect(deploy.Spec.Template.Spec.Volumes).To(ContainElement(corev1.Volume{
			Name: "contents",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "website-test-site-cm",
					},
					DefaultMode: internal.Ptr(int32(420)),
				},
			},
		}))

		By("wait for configmap to be created")
		cm := &corev1.ConfigMap{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx,
				types.NamespacedName{Name: "website-test-site-cm", Namespace: "default"},
				cm)
			return err == nil
		}, 10*time.Second, 500*time.Millisecond).Should(BeTrue())
		By("check configmap")
		Expect(cm.ObjectMeta.Name).To(Equal("website-test-site-cm"))
		Expect(cm.Data).To(HaveKeyWithValue("index.html", "test-html-content"))

		By("wait for service to be created")
		svc := &corev1.Service{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx,
				types.NamespacedName{Name: "website-test-site-service", Namespace: "default"},
				svc)
			return err == nil
		}, 10*time.Second, 500*time.Millisecond).Should(BeTrue())
		// TODO assert service object

		// --- Assert Ingress created ---
		ingress := &networkingv1.Ingress{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx,
				types.NamespacedName{Name: "website-test-site-ingress", Namespace: "default"},
				ingress)
			return err == nil
		}, 10*time.Second, 500*time.Millisecond).Should(BeTrue())
		// TODO assert ingress object
	})
})
