// Copyright Contributors to the Open Cluster Management project
package hubinfo

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	v1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers"
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

func (o *Options) validate(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("there should be no argument")
	}
	return nil
}

const (
	clusterManagerName            = "cluster-manager"
	registrationOperatorNamespace = "open-cluster-management"

	componentsNamespace                 = "open-cluster-management-hub"
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
		o.printer.Write(helpers.LEVEL_0, "Registration Operator:\t<none>\n")
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
	crdStatus := make(map[string]string)
	cmgrCrd, err := o.crdClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.TODO(), "clustermanagers.operator.open-cluster-management.io", metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if cmgrCrd != nil {
		crdStatus["clustermanagers.operator.open-cluster-management.io"] = "installed"
	} else {
		crdStatus["clustermanagers.operator.open-cluster-management.io"] = "absent"
	}

	o.printer.Write(helpers.LEVEL_0, "Registration Operator:\n")
	o.printer.Write(helpers.LEVEL_1, "Controller:\t(%d/%d) %s\n", registrationOperatorAvailableRs, registrationOperatorExpectedRs, imageName)
	o.printer.Write(helpers.LEVEL_1, "CustomResourceDefinition:\n")
	for name, st := range crdStatus {
		o.printer.Write(helpers.LEVEL_2, "(%s) %s\n", st, name)
	}
	return nil
}

func (o *Options) printComponents() error {
	cmgr, err := o.operatorClient.OperatorV1().
		ClusterManagers().
		Get(context.TODO(), clusterManagerName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		o.printer.Write(helpers.LEVEL_0, "Components:\t<uninstalled>\n")
		return nil
	}

	o.printer.Write(helpers.LEVEL_0, "Components:\n")
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
	registrationController, err := o.kubeClient.AppsV1().
		Deployments(componentsNamespace).
		Get(context.TODO(), componentNameRegistrationController, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}
	registrationWebhook, err := o.kubeClient.AppsV1().
		Deployments(componentsNamespace).
		Get(context.TODO(), componentNameRegistrationWebhook, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}

	controllerExpectedRs := int(*registrationController.Spec.Replicas)
	controllerAvailableRs := registrationController.Status.AvailableReplicas
	webhookExpectedRs := int(*registrationWebhook.Spec.Replicas)
	webhookAvailableRs := registrationWebhook.Status.AvailableReplicas
	o.printer.Write(helpers.LEVEL_1, "Registration:\n")
	o.printer.Write(helpers.LEVEL_2, "Controller:\t(%d/%d) %s\n", controllerAvailableRs, controllerExpectedRs, getImageName(registrationController))
	o.printer.Write(helpers.LEVEL_2, "Webhook:\t(%d/%d) %s\n", webhookAvailableRs, webhookExpectedRs, getImageName(registrationWebhook))
	return nil
}

func (o *Options) printWork(cmgr *v1.ClusterManager) error {
	workWebhook, err := o.kubeClient.AppsV1().
		Deployments(componentsNamespace).
		Get(context.TODO(), componentNameWorkWebhook, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}

	webhookExpectedRs := int(*workWebhook.Spec.Replicas)
	webhookAvailableRs := workWebhook.Status.AvailableReplicas
	o.printer.Write(helpers.LEVEL_1, "Work:\n")
	o.printer.Write(helpers.LEVEL_2, "Webhook:\t(%d/%d) %s\n", webhookAvailableRs, webhookExpectedRs, getImageName(workWebhook))
	return nil
}

func (o *Options) printPlacement(cmgr *v1.ClusterManager) error {
	placementController, err := o.kubeClient.AppsV1().
		Deployments(componentsNamespace).
		Get(context.TODO(), componentNamePlacementController, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}
	controllerExpectedRs := int(*placementController.Spec.Replicas)
	controllerAvailableRs := placementController.Status.AvailableReplicas
	o.printer.Write(helpers.LEVEL_1, "Placement:\n")
	o.printer.Write(helpers.LEVEL_2, "Controller:\t(%d/%d) %s\n", controllerAvailableRs, controllerExpectedRs, getImageName(placementController))
	return nil
}

func (o *Options) printComponentsCRD(cmgr *v1.ClusterManager) error {
	testingCRDNames := sets.NewString()
	for _, rs := range cmgr.Status.RelatedResources {
		if rs.Resource == "customresourcedefinitions" {
			testingCRDNames.Insert(rs.Name)
		}
	}
	crdList, err := o.crdClient.ApiextensionsV1().
		CustomResourceDefinitions().
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	statuses := make(map[string]string)
	existingCRDNames := sets.NewString()
	for _, existingCRD := range crdList.Items {
		existingCRDNames.Insert(existingCRD.Name)
	}
	for _, name := range testingCRDNames.List() {
		st := "absent"
		if existingCRDNames.Has(name) {
			st = "installed"
		}
		statuses[name] = st
	}
	o.printer.Write(helpers.LEVEL_1, "CustomResourceDefinition:\n")
	for name, st := range statuses {
		o.printer.Write(helpers.LEVEL_2, "(%s) %s\n", st, name)
	}
	return nil
}

func getImageName(deploy *appsv1.Deployment) string {
	imageName := "<none>"
	for _, container := range deploy.Spec.Template.Spec.Containers {
		imageName = container.Image
	}
	return imageName
}
