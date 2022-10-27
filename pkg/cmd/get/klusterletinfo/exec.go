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
)

func (o *Options) run() error {
	k, err := o.operatorClient.OperatorV1().Klusterlets().Get(context.TODO(), klusterletName, metav1.GetOptions{})
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
	if err := o.printRegistrationOperator(); err != nil {
		return err
	}
	// printing components
	if err := o.printComponents(k); err != nil {
		return err
	}
	return nil
}

func (o *Options) printRegistrationOperator() error {
	deploy, err := o.kubeClient.AppsV1().
		Deployments(registrationOperatorNamespace).
		Get(context.TODO(), klusterletName, metav1.GetOptions{})
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
		Get(context.TODO(), klusterletCRD, metav1.GetOptions{})
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

func (o *Options) printComponents(klet *v1.Klusterlet) error {

	o.printer.Write(printer.LEVEL_0, "Components:\n")

	if err := o.printRegistration(klet); err != nil {
		return err
	}
	if err := o.printWork(klet); err != nil {
		return err
	}
	if err := o.printComponentsCRD(klet); err != nil {
		return err
	}
	return nil
}

func (o *Options) printRegistration(klet *v1.Klusterlet) error {
	o.printer.Write(printer.LEVEL_1, "Registration:\n")
	return printer.PrintComponentsDeploy(o.printer, o.kubeClient, klet.Status.RelatedResources, componentNameRegistrationAgent)
}

func (o *Options) printWork(klet *v1.Klusterlet) error {
	o.printer.Write(printer.LEVEL_1, "Work:\n")
	return printer.PrintComponentsDeploy(o.printer, o.kubeClient, klet.Status.RelatedResources, componentNameWorkAgent)
}

func (o *Options) printComponentsCRD(klet *v1.Klusterlet) error {
	o.printer.Write(printer.LEVEL_1, "CustomResourceDefinition:\n")
	return printer.PrintComponentsCRD(o.printer, o.crdClient, klet.Status.RelatedResources)
}
