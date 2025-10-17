// Copyright Contributors to the Open Cluster Management project
package create

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonclientset "open-cluster-management.io/api/client/addon/clientset/versioned"
	workapiv1 "open-cluster-management.io/api/work/v1"

	"open-cluster-management.io/clusteradm/pkg/helpers/parse"
)

func newAddonTemplate(o *Options) (*addonv1alpha1.AddOnTemplate, error) {
	manifests, err := o.readManifests()
	if err != nil {
		return nil, err
	}

	// Parse labels
	labels, err := o.parseLabels()
	if err != nil {
		return nil, err
	}

	addon := &addonv1alpha1.AddOnTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:   o.templateName(),
			Labels: labels,
		},
		Spec: addonv1alpha1.AddOnTemplateSpec{
			AddonName: o.Name,
			AgentSpec: workapiv1.ManifestWorkSpec{
				Workload: workapiv1.ManifestsTemplate{
					Manifests: manifests,
				},
			},
			Registration: []addonv1alpha1.RegistrationSpec{},
		},
	}

	if o.EnableHubRegistration {
		addon.Spec.Registration = []addonv1alpha1.RegistrationSpec{
			{
				Type: addonv1alpha1.RegistrationTypeKubeClient,
				KubeClient: &addonv1alpha1.KubeClientRegistrationConfig{
					HubPermissions: []addonv1alpha1.HubPermissionConfig{},
				},
			},
		}

		if o.ClusterRoleBindingRef != "" {
			addon.Spec.Registration[0].KubeClient.HubPermissions = append(addon.Spec.Registration[0].KubeClient.HubPermissions,
				addonv1alpha1.HubPermissionConfig{
					Type: addonv1alpha1.HubPermissionsBindingCurrentCluster,
					CurrentCluster: &addonv1alpha1.CurrentClusterBindingConfig{
						ClusterRoleName: o.ClusterRoleBindingRef,
					},
				})
		}
	}

	return addon, nil
}

func newClusterManagementAddon(o *Options) (*addonv1alpha1.ClusterManagementAddOn, error) {
	// Parse labels
	labels, err := o.parseLabels()
	if err != nil {
		return nil, err
	}

	// Parse placement reference
	placementNamespace, placementName, err := o.parsePlacementRef()
	if err != nil {
		return nil, err
	}

	installStrategy := addonv1alpha1.InstallStrategy{
		Type: addonv1alpha1.AddonInstallStrategyManual,
	}

	// If placement-ref is provided, change install strategy to Placements
	if placementNamespace != "" && placementName != "" {
		installStrategy.Type = addonv1alpha1.AddonInstallStrategyPlacements
		installStrategy.Placements = []addonv1alpha1.PlacementStrategy{
			{
				PlacementRef: addonv1alpha1.PlacementRef{
					Namespace: placementNamespace,
					Name:      placementName,
				},
			},
		}
	}

	cma := &addonv1alpha1.ClusterManagementAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:   o.Name,
			Labels: labels,
			Annotations: map[string]string{
				"addon.open-cluster-management.io/lifecycle": "addon-manager",
			},
		},
		Spec: addonv1alpha1.ClusterManagementAddOnSpec{
			SupportedConfigs: []addonv1alpha1.ConfigMeta{
				{
					ConfigGroupResource: addonv1alpha1.ConfigGroupResource{
						Group:    addonv1alpha1.GroupVersion.Group,
						Resource: "addontemplates",
					},
					DefaultConfig: &addonv1alpha1.ConfigReferent{
						Name: o.templateName(),
					},
				},
			},
			InstallStrategy: installStrategy,
		},
	}

	return cma, nil
}

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	if len(args) == 0 {
		return fmt.Errorf("addon name must be specified")
	}

	if len(args) > 1 {
		return fmt.Errorf("only one adon name can be specified")
	}

	o.Name = args[0]

	return nil
}

func (o *Options) Validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if len(o.Version) == 0 {
		return fmt.Errorf("addon version must be specified")
	}

	if len(*o.FileNameFlags.Filenames) == 0 {
		return fmt.Errorf("manifest files must be specified")
	}

	// Validate placement-ref format if provided
	if o.PlacementRef != "" {
		_, _, err := o.parsePlacementRef()
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Options) Run() error {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}

	addonClient, err := addonclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	if err := o.applyCMA(addonClient); err != nil {
		return err
	}

	return o.applyTemplate(addonClient)
}

func (o *Options) templateName() string {
	return o.Name + "-" + o.Version
}

func (o *Options) applyCMA(addonClient addonclientset.Interface) error {
	cma, err := newClusterManagementAddon(o)
	if err != nil {
		return err
	}

	// apply cma at first
	originalCMA, err := addonClient.AddonV1alpha1().ClusterManagementAddOns().Get(context.TODO(), o.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err := addonClient.AddonV1alpha1().ClusterManagementAddOns().Create(context.TODO(), cma, metav1.CreateOptions{})
		fmt.Fprintf(o.Streams.Out, "ClusterManagementAddon %s is created\n", o.Name)
		return err
	}
	if err != nil {
		return err
	}

	if !o.Overwrite {
		fmt.Fprintf(o.Streams.Out, "ClusterManagementAddon %s is not updated when overwrite is disabled\n", o.Name)
		return nil
	}

	cma.ResourceVersion = originalCMA.ResourceVersion
	if _, err = addonClient.AddonV1alpha1().ClusterManagementAddOns().Update(context.TODO(), cma, metav1.UpdateOptions{}); err != nil {
		return err
	}

	fmt.Fprintf(o.Streams.Out, "ClusterManagementAddon %s is updated\n", o.Name)
	return nil
}

func (o *Options) applyTemplate(addonClient addonclientset.Interface) error {
	addon, err := newAddonTemplate(o)
	if err != nil {
		return err
	}

	originalAddon, err := addonClient.AddonV1alpha1().AddOnTemplates().Get(context.TODO(), o.templateName(), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err := addonClient.AddonV1alpha1().AddOnTemplates().Create(context.TODO(), addon, metav1.CreateOptions{})
		fmt.Fprintf(o.Streams.Out, "AddonTemplate %s is created\n", addon.Name)
		return err
	}
	if err != nil {
		return err
	}

	if !o.Overwrite {
		fmt.Fprintf(o.Streams.Out, "AddonTemplate %s is not updated when overwrite is disabled\n", addon.Name)
		return nil
	}

	addon.ResourceVersion = originalAddon.ResourceVersion
	if _, err = addonClient.AddonV1alpha1().AddOnTemplates().Update(context.TODO(), addon, metav1.UpdateOptions{}); err != nil {
		return err
	}

	fmt.Fprintf(o.Streams.Out, "AddonTemplate %s is updated\n", addon.Name)
	return nil
}

func (o *Options) readManifests() ([]workapiv1.Manifest, error) {
	opt := o.FileNameFlags.ToOptions()
	builder := resource.NewLocalBuilder().
		Unstructured().
		FilenameParam(false, &opt).
		Flatten().
		ContinueOnError()
	result := builder.Do()

	if err := result.Err(); err != nil {
		return nil, err
	}

	manifests := []workapiv1.Manifest{}

	items, err := result.Infos()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		manifests = append(manifests, workapiv1.Manifest{RawExtension: runtime.RawExtension{Object: item.Object}})
	}

	return manifests, nil
}

// parseLabels parses the labels flag and returns a map of labels
func (o *Options) parseLabels() (map[string]string, error) {
	return parse.ParseLabels(o.Labels)
}

// parsePlacementRef parses the placement-ref flag and returns namespace and name
func (o *Options) parsePlacementRef() (namespace, name string, err error) {
	if o.PlacementRef == "" {
		return "", "", nil
	}

	parts := strings.Split(o.PlacementRef, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("placement-ref must be in the format 'namespace/name'")
	}

	namespace = strings.TrimSpace(parts[0])
	name = strings.TrimSpace(parts[1])

	if namespace == "" || name == "" {
		return "", "", fmt.Errorf("placement-ref namespace and name cannot be empty")
	}

	return namespace, name, nil
}
