// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"fmt"
	"strings"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	hubaddoninstall "open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon"
)

func (o *Options) complete(_ *cobra.Command, _ []string) (err error) {
	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "names", o.names)
	return nil
}

func (o *Options) validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if o.names == "" {
		return fmt.Errorf("names is missing")
	}

	return nil
}

func (o *Options) run() error {
	addons := sets.New[string](strings.Split(o.names, ",")...)
	if len(addons) == 0 {
		return nil
	}

	var errs []error
	for a := range addons {
		if err := o.runWithHelmClient(a); err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

func (o *Options) runWithHelmClient(addon string) error {
	var errs []error

	for _, addonChart := range hubaddoninstall.GetAddonCharts(addon, o.namespace, "") {
		o.Helm.WithNamespace(addonChart.Namespace)

		if err := o.Helm.UninstallRelease(addonChart.ReleaseName); err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}
