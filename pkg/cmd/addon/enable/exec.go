// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonclientset "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers/parse"
)

type ClusterAddonInfo struct {
	ClusterName string
	NameSpace   string
	AddonName   string
	Annotations map[string]string
}

// applyConfigFileAndBuildReferences reads the config file, applies the resources to the cluster,
// and builds AddOnConfig references from the applied resources
func applyConfigFileAndBuildReferences(o *Options) ([]addonv1alpha1.AddOnConfig, error) {
	if o.ConfigFile == "" {
		return nil, nil
	}

	// Read the config file
	data, err := os.ReadFile(o.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", o.ConfigFile, err)
	}

	yamlReader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	var rawResources [][]byte

	for {
		doc, err := yamlReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to parse config file %s: %w", o.ConfigFile, err)
		}
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}
		rawResources = append(rawResources, doc)
	}

	if len(rawResources) == 0 {
		return nil, nil
	}

	// Apply the resources to the cluster using ResourceReader
	r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)
	if err := r.ApplyRaw(rawResources); err != nil {
		return nil, fmt.Errorf("failed to apply config resources: %w", err)
	}

	// Parse each applied resource to build AddOnConfig references
	var configs []addonv1alpha1.AddOnConfig
	restMapper, err := o.ClusteradmFlags.KubectlFactory.ToRESTMapper()
	if err != nil {
		return nil, fmt.Errorf("failed to get REST mapper: %w", err)
	}

	for _, rawResource := range rawResources {
		// Decode the YAML to unstructured object
		obj := &unstructured.Unstructured{}
		decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(rawResource)), 4096)
		if err := decoder.Decode(obj); err != nil {
			klog.Warningf("failed to decode resource: %v", err)
			continue
		}

		if obj.GetKind() == "" {
			continue
		}

		// Get the GVK from the object
		gvk := obj.GroupVersionKind()

		// Use REST mapper to get the resource name (plural form)
		mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			klog.Warningf("failed to get REST mapping for %s: %v", gvk.String(), err)
			continue
		}

		// Build AddOnConfig
		config := addonv1alpha1.AddOnConfig{
			ConfigGroupResource: addonv1alpha1.ConfigGroupResource{
				Group:    gvk.Group,
				Resource: mapping.Resource.Resource,
			},
			ConfigReferent: addonv1alpha1.ConfigReferent{
				Namespace: obj.GetNamespace(),
				Name:      obj.GetName(),
			},
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func NewClusterAddonInfo(cn string, o *Options, an string, configs []addonv1alpha1.AddOnConfig) (*addonv1alpha1.ManagedClusterAddOn, error) {
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

	// Parse provided labels
	labels, err := parse.ParseLabels(o.Labels)
	if err != nil {
		return nil, err
	}

	return &addonv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:        an,
			Namespace:   cn,
			Annotations: annos,
			Labels:      labels,
		},
		Spec: addonv1alpha1.ManagedClusterAddOnSpec{
			InstallNamespace: o.Namespace,
			Configs:          configs,
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

	// Apply config file and build AddOnConfig references once for all addons
	configs, err := applyConfigFileAndBuildReferences(o)
	if err != nil {
		return err
	}

	for _, addon := range addons {
		_, err := addonClient.AddonV1alpha1().ClusterManagementAddOns().Get(context.TODO(), addon, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("enabling the unknown addon %s is not supported", addon)
			}
			return err
		}

		for _, clusterName := range clusters {
			cai, err := NewClusterAddonInfo(clusterName, o, addon, configs)
			if err != nil {
				return err
			}
			err = ApplyAddon(addonClient, cai)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(o.Streams.Out, "Deploying %s add-on to namespaces %s of managed cluster: %s.\n",
				addon, o.Namespace, clusterName)
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
	originalAddon.Labels = addon.Labels
	originalAddon.Spec.InstallNamespace = addon.Spec.InstallNamespace
	if addon.Spec.Configs != nil {
		originalAddon.Spec.Configs = addon.Spec.Configs
	}
	_, err = addonClient.AddonV1alpha1().ManagedClusterAddOns(addon.Namespace).Update(context.TODO(), originalAddon, metav1.UpdateOptions{})
	return err
}
