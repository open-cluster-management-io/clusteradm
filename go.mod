module open-cluster-management.io/clusteradm

go 1.16

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	k8s.io/client-go => k8s.io/client-go v0.21.0
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/googleapis/gnostic v0.5.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.10 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/onsi/ginkgo v1.14.1 // indirect
	github.com/onsi/gomega v1.10.2 // indirect
	github.com/openshift/library-go v0.0.0-20210521084623-7392ea9b02ca
	github.com/spf13/cobra v1.1.3
	google.golang.org/appengine v1.6.6 // indirect
	k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/cli-runtime v0.21.0
	k8s.io/client-go v0.21.1
	k8s.io/component-base v0.21.1
	k8s.io/klog v1.0.0
	k8s.io/kubectl v0.21.0
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
	open-cluster-management.io/api v0.0.0-20210519100835-bb9d4652f920
)
