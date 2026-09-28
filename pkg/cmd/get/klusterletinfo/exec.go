// Copyright Contributors to the Open Cluster Management project
package klusterletinfo

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
	err = o.ClusteradmFlags.ValidateManagedCluster()
	if err != nil {
		return err
	}

	if len(args) != 0 {
		return fmt.Errorf("there should be no argument")
	}

	return nil
}

const (
	klusterletName                = "klusterlet"
	registrationOperatorNamespace = "open-cluster-management"
	klusterletCRD                 = "klusterlets.operator.open-cluster-management.io"

	componentNameRegistrationAgent = "klusterlet-registration-agent"
	componentNameWorkAgent         = "klusterlet-work-agent"
	componentNameKlusterletAgent   = "klusterlet-agent"
)

func (o *Options) run(ctx context.Context) error {
	k, err := o.operatorClient.OperatorV1().Klusterlets().Get(ctx, klusterletName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		o.printer.Write(printer.LEVEL_0, "No klusterlet detected! Please make sure you're using the correct context!\n")
		return nil
	}

	o.printer.Write(printer.LEVEL_0, "Klusterlet Conditions:\n")
	for _, v := range k.Status.Conditions {
		o.printer.Write(printer.LEVEL_1, "Type:\t\t\t%v\n", v.Type)
		o.printer.Write(printer.LEVEL_1, "Status:\t\t%v\n", v.Status)
		o.printer.Write(printer.LEVEL_1, "LastTransitionTime:\t%v\n", v.LastTransitionTime)
		o.printer.Write(printer.LEVEL_1, "Reason:\t\t%v\n", v.Reason)
		o.printer.Write(printer.LEVEL_1, "Message:\t\t%v\n", v.Message)
		o.printer.Write(printer.LEVEL_0, "\n")
	}

	// printing registration-operator
	if err := o.printRegistrationOperator(ctx); err != nil {
		return err
	}
	// printing components
	if err := o.printComponents(ctx, k); err != nil {
		return err
	}
	return nil
}

func (o *Options) printRegistrationOperator(ctx context.Context) error {
	deploy, err := o.kubeClient.AppsV1().
		Deployments(registrationOperatorNamespace).
		Get(ctx, klusterletName, metav1.GetOptions{})
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
		if container.Name == "klusterlet" {
			imageName = container.Image
		}
	}
	crdStatus := make(map[string]string)
	cmgrCrd, err := o.crdClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(ctx, klusterletCRD, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if cmgrCrd != nil {
		crdStatus[klusterletCRD] = "installed"
	} else {
		crdStatus[klusterletCRD] = "absent"
	}

	o.printer.Write(printer.LEVEL_0, "Registration Operator:\n")
	o.printer.Write(printer.LEVEL_1, "Controller:\t(%d/%d) %s\n", registrationOperatorAvailableRs, registrationOperatorExpectedRs, imageName)
	o.printer.Write(printer.LEVEL_1, "CustomResourceDefinition:\n")
	for name, st := range crdStatus {
		o.printer.Write(printer.LEVEL_2, "(%s) %s\n", st, name)
	}
	return nil
}

func (o *Options) printComponents(ctx context.Context, klet *v1.Klusterlet) error {
	o.printer.Write(printer.LEVEL_0, "Components:\n")

	mode := klet.Spec.DeployOption.Mode
	if mode == v1.InstallModeSingleton || mode == v1.InstallModeSingletonHosted {
		if err := o.printAgent(ctx, klet); err != nil {
			return err
		}
	} else {
		if err := o.printRegistration(ctx, klet); err != nil {
			return err
		}
		if err := o.printWork(ctx, klet); err != nil {
			return err
		}
	}
	if err := o.printComponentsCRD(ctx, klet); err != nil {
		return err
	}
	return nil
}

func (o *Options) printRegistration(ctx context.Context, klet *v1.Klusterlet) error {
	o.printer.Write(printer.LEVEL_1, "Registration:\n")
	return printer.PrintComponentsDeploy(ctx, o.printer, o.kubeClient, klet.Status.RelatedResources, componentNameRegistrationAgent)
}

func (o *Options) printWork(ctx context.Context, klet *v1.Klusterlet) error {
	o.printer.Write(printer.LEVEL_1, "Work:\n")
	return printer.PrintComponentsDeploy(ctx, o.printer, o.kubeClient, klet.Status.RelatedResources, componentNameWorkAgent)
}

func (o *Options) printAgent(ctx context.Context, klet *v1.Klusterlet) error {
	o.printer.Write(printer.LEVEL_1, "Controller:\n")
	return printer.PrintComponentsDeploy(ctx, o.printer, o.kubeClient, klet.Status.RelatedResources, componentNameKlusterletAgent)
}

func (o *Options) printComponentsCRD(ctx context.Context, klet *v1.Klusterlet) error {
	o.printer.Write(printer.LEVEL_1, "CustomResourceDefinition:\n")
	return printer.PrintComponentsCRD(ctx, o.printer, o.crdClient, klet.Status.RelatedResources)
}
