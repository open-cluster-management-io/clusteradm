// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonclientset "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
)

type ClusterAddonInfo struct {
	ClusterName string
	NameSpace   string
	AddonName   string
	Annotations map[string]string
}

func NewClusterAddonInfo(cn string, o *Options, an string) (*addonv1alpha1.ManagedClusterAddOn, error) {
	// Parse provided annotations
	annos := map[string]string{}
	for _, annoString := range o.Annotate {
		annoSlice := strings.Split(annoString, "=")
		if len(annoSlice) != 2 {
			return nil,
				fmt.Errorf("error parsing annotation '%s'. Expected to be of the form: key=value", annoString)
		}
		annos[annoSlice[0]] = annoSlice[1]
	}
	return &addonv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:        an,
			Namespace:   cn,
			Annotations: annos,
		},
		Spec: addonv1alpha1.ManagedClusterAddOnSpec{
			InstallNamespace: o.Namespace,
		},
	}, nil
}

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("enable options:", "dry-run", o.ClusteradmFlags.DryRun, "names", o.Names, "clusters", o.ClusterOptions.AllClusters(), "output-file", o.OutputFile)

	return nil
}

func (o *Options) Validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if len(o.Names) == 0 {
		return fmt.Errorf("names is missing")
	}

	if err := o.ClusterOptions.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *Options) Run() error {
	addons := sets.NewString(o.Names...)
	clusters := o.ClusterOptions.AllClusters()

	klog.V(3).InfoS("values:", "addon", addons, "clusters", clusters)

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	addonClient, err := addonclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	return o.runWithClient(clusterClient, addonClient, addons.List(), clusters.UnsortedList())
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface,
	addonClient addonclientset.Interface,
	addons []string,
	clusters []string) error {

	for _, clusterName := range clusters {
		_, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(),
			clusterName,
			metav1.GetOptions{})
		if err != nil {
			return err
		}
	}

	for _, addon := range addons {
		for _, clusterName := range clusters {
			cai, err := NewClusterAddonInfo(clusterName, o, addon)
			if err != nil {
				return err
			}
			err = ApplyAddon(addonClient, cai)
			if err != nil {
				return err
			}

			fmt.Fprintf(o.Streams.Out, "Deploying %s add-on to namespaces %s of managed cluster: %s.\n", addon, o.Namespace, clusterName)
		}
	}

	return nil
}

func ApplyAddon(addonClient addonclientset.Interface, addon *addonv1alpha1.ManagedClusterAddOn) error {
	originalAddon, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(addon.Namespace).Get(context.TODO(), addon.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(addon.Namespace).Create(context.TODO(), addon, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	originalAddon.Annotations = addon.Annotations
	originalAddon.Spec.InstallNamespace = addon.Spec.InstallNamespace
	_, err = addonClient.AddonV1alpha1().ManagedClusterAddOns(addon.Namespace).Update(context.TODO(), originalAddon, metav1.UpdateOptions{})
	return err
}
