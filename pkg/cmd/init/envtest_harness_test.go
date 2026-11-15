// Copyright Contributors to the Open Cluster Management project
package init

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// The tests that need an API server share one envtest control plane, started on first use and
// skipped when the envtest binaries aren't installed (KUBEBUILDER_ASSETS).
var (
	envOnce sync.Once
	env     *envtest.Environment
	envCfg  *rest.Config
	envErr  error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if env != nil {
		_ = env.Stop()
	}
	os.Exit(code)
}

func testEnvConfig(t *testing.T) *rest.Config {
	t.Helper()
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("envtest binaries are not available: set KUBEBUILDER_ASSETS")
	}
	envOnce.Do(func() {
		env = &envtest.Environment{
			ErrorIfCRDPathMissing: true,
			CRDDirectoryPaths: []string{
				filepath.Join("..", "..", "..", "vendor", "open-cluster-management.io", "api", "cluster", "v1"),
			},
		}
		envCfg, envErr = env.Start()
	})
	if envErr != nil {
		t.Fatal(envErr)
	}
	return envCfg
}

// envFlags returns clusteradm flags whose factory talks to cfg.
func envFlags(cfg *rest.Config) *genericclioptionsclusteradm.ClusteradmFlags {
	return genericclioptionsclusteradm.NewClusteradmFlags(cmdutil.NewFactory(clientGetter{cfg: cfg}))
}

type clientGetter struct{ cfg *rest.Config }

func (g clientGetter) ToRESTConfig() (*rest.Config, error) { return rest.CopyConfig(g.cfg), nil }

func (g clientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	d, err := discovery.NewDiscoveryClientForConfig(g.cfg)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(d), nil
}

func (g clientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	d, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(d), nil
}

func (g clientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	c := clientcmdapi.NewConfig()
	c.Clusters["env"] = &clientcmdapi.Cluster{Server: g.cfg.Host, CertificateAuthorityData: g.cfg.CAData}
	c.AuthInfos["env"] = &clientcmdapi.AuthInfo{ClientCertificateData: g.cfg.CertData, ClientKeyData: g.cfg.KeyData,
		Token: g.cfg.BearerToken}
	c.Contexts["env"] = &clientcmdapi.Context{Cluster: "env", AuthInfo: "env"}
	c.CurrentContext = "env"
	return clientcmd.NewDefaultClientConfig(*c, nil)
}
