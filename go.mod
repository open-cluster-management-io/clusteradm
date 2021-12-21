module open-cluster-management.io/clusteradm

go 1.16

replace github.com/go-logr/logr => github.com/go-logr/logr v0.4.0

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/briandowns/spinner v1.11.1
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/library-go v0.0.0-20210916194400-ae21aab32431
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/cli-runtime v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/component-base v0.22.1
	k8s.io/klog/v2 v2.9.0
	k8s.io/kubectl v0.21.0
	open-cluster-management.io/api v0.0.0-20210927063308-2c6896161c48
	sigs.k8s.io/controller-runtime v0.8.3
)
