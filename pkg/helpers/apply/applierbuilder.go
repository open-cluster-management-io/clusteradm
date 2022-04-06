// Copyright Contributors to the Open Cluster Management project
package apply

import (
	"text/template"

	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Applier struct {
	kubeClient          kubernetes.Interface
	apiExtensionsClient apiextensionsclient.Interface
	dynamicClient       dynamic.Interface
	templateFuncMap     template.FuncMap
	scheme              *runtime.Scheme
	owner               runtime.Object
	cache               resourceapply.ResourceCache
	controller          *bool
	blockOwnerDeletion  *bool
}

//ApplierBuilder a builder to build the applier
type ApplierBuilder struct {
	Applier
}

type iApplierBuilder interface {
	//Build returns the builded applier
	Build() Applier
	//WithClient adds the several clients to the applier
	WithClient(
		kubeClient kubernetes.Interface,
		apiExtensionsClient apiextensionsclient.Interface,
		dynamicClient dynamic.Interface) *ApplierBuilder
	//WithTemplateFuncMap add template.FuncMap to the applier.
	WithTemplateFuncMap(fm template.FuncMap) *ApplierBuilder
	//WithOwner add an ownerref to the object
	WithOwner(owner runtime.Object, blockOwnerDeletion, controller bool, scheme *runtime.Scheme) *ApplierBuilder
}

var _ iApplierBuilder = &ApplierBuilder{}

//Build returns the builded applier
func (a *ApplierBuilder) Build() Applier {
	a.cache = resourceapply.NewResourceCache()
	return a.Applier
}

//WithClient adds the several clients to the applier
func (a *ApplierBuilder) WithClient(
	kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface) *ApplierBuilder {
	a.kubeClient = kubeClient
	a.apiExtensionsClient = apiExtensionsClient
	a.dynamicClient = dynamicClient
	return a
}

//WithTemplateFuncMap add template.FuncMap to the applier.
func (a *ApplierBuilder) WithTemplateFuncMap(fm template.FuncMap) *ApplierBuilder {
	a.templateFuncMap = fm
	return a
}

//WithOwner add an ownerref to the object
func (a *ApplierBuilder) WithOwner(owner runtime.Object, blockOwnerDeletion, controller bool, scheme *runtime.Scheme) *ApplierBuilder {
	a.owner = owner
	a.blockOwnerDeletion = &blockOwnerDeletion
	a.controller = &controller
	a.scheme = scheme
	return a
}
