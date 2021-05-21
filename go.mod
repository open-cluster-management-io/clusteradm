module open-cluster-management.io/clusteradm

go 1.16

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	k8s.io/client-go => k8s.io/client-go v0.20.4
)

require (
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/onsi/gomega v1.10.1 // indirect
	github.com/spf13/cobra v1.1.3
	k8s.io/cli-runtime v0.20.5
	k8s.io/client-go v1.5.2
	k8s.io/component-base v0.20.1
	k8s.io/kubectl v0.20.1
)
