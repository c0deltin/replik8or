package source

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/c0deltin/replik8or/internal/config"
	"github.com/c0deltin/replik8or/internal/replicator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	ctx       context.Context
	cncl      context.CancelFunc
	k8sClient ctrlclient.Client
	testEnv   *envtest.Environment

	systemNamespaces  = []string{"kube-node-lease", "kube-public", "kube-system"}
	replicaNamespaces = []string{"testing", "foo", "bar"}
	newNamespace      = "new"
	sourceNamespace   = "default"

	sourceConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "source-configmap",
			Namespace: sourceNamespace,
			Annotations: map[string]string{
				replicator.ReplicationAllowedAnnotation: "true",
			},
		},
		Data: map[string]string{
			"foo": "bar",
		},
	}
)

func TestSourceReconciler(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "SourceReconciler Suite")
}

var _ = BeforeSuite(func() {
	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cncl = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		BinaryAssetsDirectory: filepath.Join(
			"..",
			"..",
			"..",
			"bin",
			"k8s",
			fmt.Sprintf("1.34.0-%s-%s", runtime.GOOS, runtime.GOARCH),
		),
	}

	restCfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(restCfg).NotTo(BeNil())

	k8sClient, err = ctrlclient.New(restCfg, ctrlclient.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(restCfg, ctrl.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	err = NewReconciler[*corev1.ConfigMap](
		k8sClient,
		&config.Config{DisallowedNamespaces: systemNamespaces},
		replicator.EmptyConfigMap,
		replicator.EmptyConfigMapList,
	).SetupWithManager("ConfigMap", k8sManager)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).NotTo(HaveOccurred(), "failed to start manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cncl()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
