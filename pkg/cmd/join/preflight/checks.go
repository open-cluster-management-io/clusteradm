// Copyright Contributors to the Open Cluster Management project
package preflight

import (
	"strings"

	"github.com/pkg/errors"
)

type KlusterletApiserverCheck struct {
	KlusterletApiserver string
}

func (c KlusterletApiserverCheck) Check() (warnings []error, errorList []error) {
	if !validAPIHost(c.KlusterletApiserver) {
		return nil, []error{errors.New("ConfigMap/cluster-info.data.kubeconfig.clusters[0].cluster.server field in namespace kube-public should start with http:// or https://, please edit it first")}
	}
	return nil, nil
}

func (c KlusterletApiserverCheck) Name() string {
	return "KlusterletApiserver check"
}

func validAPIHost(host string) bool {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return true
	}
	return false
}
