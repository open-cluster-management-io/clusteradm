// Copyright Red Hat
package apply

import (
	"context"
	"text/template"

	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Applier struct {
	kubeClient          kubernetes.Interface
	apiExtensionsClient apiextensionsclient.Interface
	dynamicClient       dynamic.Interface
	templateFuncMap     template.FuncMap
	scheme              *runtime.Scheme
	owner               runtime.Object
	cache               resourceapply.ResourceCache
	context             context.Context
	controller          *bool
	blockOwnerDeletion  *bool
	kindOrder           KindsOrder
}

// ApplierBuilder a builder to build the applier
type ApplierBuilder struct {
	applier Applier
}

type iApplierBuilder interface {
	// Build returns the builded applier
	Build() Applier
	// WithRestConfig adds the necessary clients using a rest.Config
	WithRestConfig(cfg *rest.Config) *ApplierBuilder
	// WithClient adds the several clients to the applier
	WithClient(
		kubeClient kubernetes.Interface,
		apiExtensionsClient apiextensionsclient.Interface,
		dynamicClient dynamic.Interface) *ApplierBuilder
	// WithTemplateFuncMap add template.FuncMap to the applier.
	WithTemplateFuncMap(fm template.FuncMap) *ApplierBuilder
	// WithOwner add an ownerref to the object
	WithOwner(owner runtime.Object, blockOwnerDeletion, controller bool, scheme *runtime.Scheme) *ApplierBuilder
	// WithCache add cache
	WithCache(cache resourceapply.ResourceCache) *ApplierBuilder
	// WithContext use a context or use a new one if not provided
	WithContext(ctx context.Context) *ApplierBuilder
	// WithKindOrder define in which order to the files must be applied
	WithKindOrder(kindOrder KindsOrder) *ApplierBuilder
	// GetKubeClient returns the kubeclient
	GetKubeClient() kubernetes.Interface
	// GetAPIExtensionClient returns the APIExtensionClient
	GetAPIExtensionClient() apiextensionsclient.Interface
	// GetDynamicClient returns the dynamicClient
	GetDynamicClient() dynamic.Interface
}

var _ iApplierBuilder = NewApplierBuilder()

// New ApplyBuilder
func NewApplierBuilder() *ApplierBuilder {
	return &ApplierBuilder{}
}

// Build returns the builded applier
func (a *ApplierBuilder) Build() Applier {
	if a.applier.cache == nil {
		a.applier.cache = NewResourceCache()
	}
	if a.applier.context == nil {
		a.applier.context = context.Background()
	}
	if a.applier.kindOrder == nil {
		a.applier.kindOrder = DefaultCreateUpdateKindsOrder
	}
	return a.applier
}

// WithRestConfig adds the clients based on the provided rest.Config
func (a *ApplierBuilder) WithRestConfig(cfg *rest.Config) *ApplierBuilder {
	kubeClient := kubernetes.NewForConfigOrDie(cfg)
	apiExtensionsClient := apiextensionsclient.NewForConfigOrDie(cfg)
	dynamicClient := dynamic.NewForConfigOrDie(cfg)

	a.applier.kubeClient = kubeClient
	a.applier.apiExtensionsClient = apiExtensionsClient
	a.applier.dynamicClient = dynamicClient
	return a
}

// WithClient adds the several clients to the applier
func (a *ApplierBuilder) WithClient(
	kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface) *ApplierBuilder {
	a.applier.kubeClient = kubeClient
	a.applier.apiExtensionsClient = apiExtensionsClient
	a.applier.dynamicClient = dynamicClient
	return a
}

// WithTemplateFuncMap add template.FuncMap to the applier.
func (a *ApplierBuilder) WithTemplateFuncMap(fm template.FuncMap) *ApplierBuilder {
	a.applier.templateFuncMap = fm
	return a
}

// WithOwner add an ownerref to the object
func (a *ApplierBuilder) WithOwner(owner runtime.Object,
	blockOwnerDeletion,
	controller bool,
	scheme *runtime.Scheme) *ApplierBuilder {
	a.applier.owner = owner
	a.applier.blockOwnerDeletion = &blockOwnerDeletion
	a.applier.controller = &controller
	a.applier.scheme = scheme
	return a
}

// WithCache set a the cache instead of using the default cache created on the Build()
func (a *ApplierBuilder) WithCache(cache resourceapply.ResourceCache) *ApplierBuilder {
	a.applier.cache = cache
	return a
}

// WithContext  set a the cache instead of using the default cache created on the Build()
func (a *ApplierBuilder) WithContext(ctx context.Context) *ApplierBuilder {
	a.applier.context = ctx
	return a
}

// WithKindOrder defines the order in which the files must be applied.
func (a *ApplierBuilder) WithKindOrder(kindsOrder KindsOrder) *ApplierBuilder {
	a.applier.kindOrder = kindsOrder
	return a
}

func (a *ApplierBuilder) GetKubeClient() kubernetes.Interface {
	return a.applier.kubeClient
}

func (a *ApplierBuilder) GetAPIExtensionClient() apiextensionsclient.Interface {
	return a.applier.apiExtensionsClient
}

func (a *ApplierBuilder) GetDynamicClient() dynamic.Interface {
	return a.applier.dynamicClient
}

func NewResourceCache() resourceapply.ResourceCache {
	return resourceapply.NewResourceCache()
}
