// Copyright Contributors to the Open Cluster Management project
package hubinfo

import (
	"context"
	"fmt"

	"open-cluster-management.io/api/feature"
	"open-cluster-management.io/clusteradm/pkg/helpers/check"

	"github.com/spf13/cobra"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	v1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

func (o *Options) complete(_ *cobra.Command, _ []string) error {
	cfg, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	operatorClient, err := operatorclient.NewForConfig(cfg)
	if err != nil {
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	crdClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	o.kubeClient = kubeClient
	o.operatorClient = operatorClient
	o.crdClient = crdClient
	return nil
}

func (o *Options) validate(args []string) (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if len(args) != 0 {
		return fmt.Errorf("there should be no argument")
	}

	return nil
}

const (
	clusterManagerName            = "cluster-manager"
	registrationOperatorNamespace = "open-cluster-management"
	clusterManagerNameCRD         = "clustermanagers.operator.open-cluster-management.io"

	componentNameRegistrationController = "cluster-manager-registration-controller"
	componentNameRegistrationWebhook    = "cluster-manager-registration-webhook"
	componentNameWorkController         = "cluster-manager-work-controller"
	componentNameWorkWebhook            = "cluster-manager-work-webhook"
	componentNamePlacementController    = "cluster-manager-placement-controller"
	componentNameAddOnManagerController = "cluster-manager-addon-manager-controller"
)

func (o *Options) run(ctx context.Context) error {
	// printing registration-operator
	if err := o.printRegistrationOperator(ctx); err != nil {
		return err
	}
	// printing components
	if err := o.printComponents(ctx); err != nil {
		return err
	}
	return nil
}

func (o *Options) printRegistrationOperator(ctx context.Context) error {
	deploy, err := o.kubeClient.AppsV1().
		Deployments(registrationOperatorNamespace).
		Get(ctx, clusterManagerName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		o.printer.Write(printer.LEVEL_0, "Registration Operator:\t<none>\n")
		return nil
	}
	imageName := "<none>"
	registrationOperatorExpectedRs := *deploy.Spec.Replicas
	registrationOperatorAvailableRs := deploy.Status.AvailableReplicas
	for _, container := range deploy.Spec.Template.Spec.Containers {
		if container.Name == "registration-operator" {
			imageName = container.Image
		}
	}

	o.printer.Write(printer.LEVEL_0, "Registration Operator:\n")
	o.printer.Write(printer.LEVEL_1, "Controller:\t(%d/%d) %s\n", registrationOperatorAvailableRs, registrationOperatorExpectedRs, imageName)
	o.printer.Write(printer.LEVEL_1, "CustomResourceDefinition:\n")
	return printer.PrintOperatorCRD(ctx, o.printer, o.crdClient, clusterManagerNameCRD)
}

func (o *Options) printComponents(ctx context.Context) error {
	cmgr, err := o.operatorClient.OperatorV1().
		ClusterManagers().
		Get(ctx, clusterManagerName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		o.printer.Write(printer.LEVEL_0, "Components:\t<uninstalled>\n")
		return nil
	}

	o.printer.Write(printer.LEVEL_0, "Components:\n")

	if err := o.printAddOnManager(ctx, cmgr); err != nil {
		return err
	}
	if err := o.printRegistration(ctx, cmgr); err != nil {
		return err
	}
	if err := o.printWork(ctx, cmgr); err != nil {
		return err
	}
	if err := o.printPlacement(ctx, cmgr); err != nil {
		return err
	}
	if err := o.printComponentsCRD(ctx, cmgr); err != nil {
		return err
	}
	return nil
}

func (o *Options) printRegistration(ctx context.Context, cmgr *v1.ClusterManager) error {
	o.printer.Write(printer.LEVEL_1, "Registration:\n")
	err := printer.PrintComponentsDeploy(ctx, o.printer, o.kubeClient, cmgr.Status.RelatedResources, componentNameRegistrationController)
	if err != nil {
		return err
	}

	return printer.PrintComponentsDeploy(ctx, o.printer, o.kubeClient, cmgr.Status.RelatedResources, componentNameRegistrationWebhook)
}

func (o *Options) printWork(ctx context.Context, cmgr *v1.ClusterManager) error {
	o.printer.Write(printer.LEVEL_1, "Work:\n")
	if cmgr.Spec.WorkConfiguration != nil && check.IsFeatureEnabled(cmgr.Spec.WorkConfiguration.FeatureGates, string(feature.ManifestWorkReplicaSet)) {
		err := printer.PrintComponentsDeploy(ctx, o.printer, o.kubeClient, cmgr.Status.RelatedResources, componentNameWorkController)
		if err != nil {
			return err
		}
	}
	return printer.PrintComponentsDeploy(ctx, o.printer, o.kubeClient, cmgr.Status.RelatedResources, componentNameWorkWebhook)
}

func (o *Options) printPlacement(ctx context.Context, cmgr *v1.ClusterManager) error {
	o.printer.Write(printer.LEVEL_1, "Placement:\n")
	return printer.PrintComponentsDeploy(ctx, o.printer, o.kubeClient, cmgr.Status.RelatedResources, componentNamePlacementController)
}

func (o *Options) printAddOnManager(ctx context.Context, cmgr *v1.ClusterManager) error {
	if cmgr.Spec.AddOnManagerConfiguration != nil && !check.IsFeatureEnabled(cmgr.Spec.AddOnManagerConfiguration.FeatureGates, string(feature.AddonManagement)) {
		return nil
	}
	o.printer.Write(printer.LEVEL_1, "AddOn Manager:\n")
	return printer.PrintComponentsDeploy(ctx, o.printer, o.kubeClient, cmgr.Status.RelatedResources, componentNameAddOnManagerController)
}

func (o *Options) printComponentsCRD(ctx context.Context, cmgr *v1.ClusterManager) error {
	o.printer.Write(printer.LEVEL_1, "CustomResourceDefinition:\n")
	return printer.PrintComponentsCRD(ctx, o.printer, o.crdClient, cmgr.Status.RelatedResources)
}
