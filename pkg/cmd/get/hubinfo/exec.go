// Copyright Contributors to the Open Cluster Management project
package hubinfo

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	v1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

func (o *Options) complete(cmd *cobra.Command, args []string) error {
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
	componentNameWorkWebhook            = "cluster-manager-work-webhook"
	componentNamePlacementController    = "cluster-manager-placement-controller"
)

func (o *Options) run() error {
	// printing registration-operator
	if err := o.printRegistrationOperator(); err != nil {
		return err
	}
	// printing components
	if err := o.printComponents(); err != nil {
		return err
	}
	return nil
}

func (o *Options) printRegistrationOperator() error {
	deploy, err := o.kubeClient.AppsV1().
		Deployments(registrationOperatorNamespace).
		Get(context.TODO(), clusterManagerName, metav1.GetOptions{})
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
	return printer.PrintOperatorCRD(o.printer, o.crdClient, clusterManagerNameCRD)
}

func (o *Options) printComponents() error {
	cmgr, err := o.operatorClient.OperatorV1().
		ClusterManagers().
		Get(context.TODO(), clusterManagerName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		o.printer.Write(printer.LEVEL_0, "Components:\t<uninstalled>\n")
		return nil
	}

	o.printer.Write(printer.LEVEL_0, "Components:\n")
	if err := o.printRegistration(cmgr); err != nil {
		return err
	}
	if err := o.printWork(cmgr); err != nil {
		return err
	}
	if err := o.printPlacement(cmgr); err != nil {
		return err
	}
	if err := o.printComponentsCRD(cmgr); err != nil {
		return err
	}
	return nil
}

func (o *Options) printRegistration(cmgr *v1.ClusterManager) error {
	o.printer.Write(printer.LEVEL_1, "Registration:\n")
	err := printer.PrintComponentsDeploy(o.printer, o.kubeClient, cmgr.Status.RelatedResources, componentNameRegistrationController)
	if err != nil {
		return err
	}

	return printer.PrintComponentsDeploy(o.printer, o.kubeClient, cmgr.Status.RelatedResources, componentNameRegistrationWebhook)
}

func (o *Options) printWork(cmgr *v1.ClusterManager) error {
	o.printer.Write(printer.LEVEL_1, "Work:\n")
	return printer.PrintComponentsDeploy(o.printer, o.kubeClient, cmgr.Status.RelatedResources, componentNameWorkWebhook)
}

func (o *Options) printPlacement(cmgr *v1.ClusterManager) error {
	o.printer.Write(printer.LEVEL_1, "Placement:\n")
	return printer.PrintComponentsDeploy(o.printer, o.kubeClient, cmgr.Status.RelatedResources, componentNamePlacementController)
}

func (o *Options) printComponentsCRD(cmgr *v1.ClusterManager) error {
	o.printer.Write(printer.LEVEL_1, "CustomResourceDefinition:\n")
	return printer.PrintComponentsCRD(o.printer, o.crdClient, cmgr.Status.RelatedResources)
}
