module open-cluster-management.io/clusteradm

go 1.16

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	k8s.io/client-go => k8s.io/client-go v0.20.4
)

require (
	github.com/open-cluster-management/cm-cli v0.0.0-20210519115358-be3bb81f33d0
	github.com/spf13/cobra v1.1.3
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/cli-runtime v0.20.5
	k8s.io/client-go v1.5.2
	k8s.io/component-base v0.20.2
	k8s.io/kubectl v0.20.1
	sigs.k8s.io/controller-runtime v0.8.3
)
