module github.com/open-cluster-management/cm-cli

go 1.15

replace (
	// github.com/open-cluster-management/library-go => /Users/dvernier/acm/library-go
	k8s.io/client-go => k8s.io/client-go v0.20.4
)

require (
	github.com/ghodss/yaml v1.0.0
	github.com/open-cluster-management/library-go v0.0.0-20210310132350-10c402d74b0a
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/cli-runtime v0.20.4
	k8s.io/client-go v1.5.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.6.2
)
