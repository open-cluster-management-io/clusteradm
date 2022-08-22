// Copyright Red Hat
package apply

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/stolostron/applier/pkg/asset"
	"github.com/stolostron/applier/pkg/helpers"

	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
)

var (
	genericScheme = runtime.NewScheme()
	genericCodecs = serializer.NewCodecFactory(genericScheme)
	genericCodec  = genericCodecs.UniversalDeserializer()
)

// WithRestConfig adds the clients based on the provided rest.Config
func (a Applier) WithRestConfig(cfg *rest.Config) Applier {
	applier := a
	kubeClient := kubernetes.NewForConfigOrDie(cfg)
	apiExtensionsClient := apiextensionsclient.NewForConfigOrDie(cfg)
	dynamicClient := dynamic.NewForConfigOrDie(cfg)

	applier.kubeClient = kubeClient
	applier.apiExtensionsClient = apiExtensionsClient
	applier.dynamicClient = dynamicClient
	return applier
}

// WithClient adds the several clients to the applier
func (a Applier) WithClient(
	kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface) Applier {
	applier := a
	applier.kubeClient = kubeClient
	applier.apiExtensionsClient = apiExtensionsClient
	applier.dynamicClient = dynamicClient
	return applier
}

// WithTemplateFuncMap add template.FuncMap to the applier.
func (a Applier) WithTemplateFuncMap(fm template.FuncMap) Applier {
	applier := a
	applier.templateFuncMap = fm
	return applier
}

// WithOwner add an ownerref to the object
func (a Applier) WithOwner(owner runtime.Object,
	blockOwnerDeletion,
	controller bool,
	scheme *runtime.Scheme) Applier {
	applier := a
	applier.owner = owner
	applier.blockOwnerDeletion = &blockOwnerDeletion
	applier.controller = &controller
	applier.scheme = scheme
	return applier
}

// WithCache set a the cache instead of using the default cache created on the Build()
func (a Applier) WithCache(cache resourceapply.ResourceCache) Applier {
	applier := a
	applier.cache = cache
	return applier
}

// WithContext  set a the cache instead of using the default cache created on the Build()
func (a Applier) WithContext(ctx context.Context) Applier {
	applier := a
	applier.context = ctx
	return applier
}

// WithKindOrder defines the order in which the files must be applied.
func (a Applier) WithKindOrder(kindsOrder KindsOrder) Applier {
	applier := a
	applier.kindOrder = kindsOrder
	return applier
}

func (a Applier) GetCache() resourceapply.ResourceCache {
	return a.cache
}

func (a Applier) Apply(reader asset.ScenarioReader,
	values interface{},
	dryRun bool,
	headerFile string,
	files ...string) ([]string, error) {
	var err error
	var memFSReader asset.ScenarioReader
	memFSReader, files, err = getFiles(reader, files, headerFile)
	if err != nil {
		return nil, err
	}

	filesInfo, err := a.GetFileInfo(memFSReader, values, headerFile, files...)
	if err != nil {
		return nil, err
	}
	output := make([]string, 0)
	filesDirectly, filesCustomResource, filesDeployment := splitFiles(files, filesInfo)
	if len(filesDirectly) != 0 {
		out, err := a.ApplyDirectly(memFSReader, values, dryRun, headerFile, filesDirectly...)
		if err != nil {
			return output, err
		}
		output = append(output, out...)
	}
	if len(filesCustomResource) != 0 {
		out, err := a.ApplyCustomResources(memFSReader, values, dryRun, headerFile, filesCustomResource...)
		if err != nil {
			return output, err
		}
		output = append(output, out...)
	}
	if len(filesDeployment) != 0 {
		out, err := a.ApplyDeployments(memFSReader, values, dryRun, headerFile, filesDeployment...)
		if err != nil {
			return output, err
		}
		output = append(output, out...)
	}
	return output, nil
}

func splitFiles(files []string, filesInfo []FileInfo) (filesDirectly,
	filesCustomResource,
	filesDeployment []string) {
	filesDirectly = make([]string, 0)
	filesCustomResource = make([]string, 0)
	filesDeployment = make([]string, 0)
	for _, fileInfo := range filesInfo {
		resourceVersion := strings.Split(fileInfo.APIVersion, "/")
		if len(resourceVersion) == 1 {
			filesDirectly = append(filesDirectly, fileInfo.FileName)
			continue
		}
		if resourceVersion[0] == "apps" && fileInfo.Kind == "Deployment" {
			filesDeployment = append(filesDeployment, fileInfo.FileName)
			continue
		}
		filesCustomResource = append(filesCustomResource, fileInfo.FileName)
	}
	return
}

//ApplyDeployments applies a appsv1.Deployment template
func (a Applier) ApplyDeployments(
	reader asset.ScenarioReader,
	values interface{},
	dryRun bool,
	headerFile string,
	files ...string) ([]string, error) {
	output := make([]string, 0)
	// Remove header files from the files as it should not be processed.
	files = asset.Delete(files, headerFile)
	//Render each file
	for _, name := range files {
		if name == headerFile {
			continue
		}
		deployment, err := a.ApplyDeployment(reader, values, dryRun, headerFile, name)
		if err != nil {
			if helpers.IsEmptyAsset(err) {
				continue
			}
			return output, err
		}
		output = append(output, deployment)
	}
	return output, nil
}

//ApplyDeployment apply a deployment
func (a Applier) ApplyDeployment(
	reader asset.ScenarioReader,
	values interface{},
	dryRun bool,
	headerFile string,
	name string) (string, error) {
	genericScheme.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	recorder := events.NewInMemoryRecorder(helpers.GetExampleHeader())
	deploymentBytes, err := a.MustTemplateAsset(reader, values, headerFile, name)
	if err != nil {
		return string(deploymentBytes), err
	}
	output := string(deploymentBytes)
	if dryRun {
		return output, nil
	}
	deployment, sch, err := genericCodec.Decode(deploymentBytes, nil, nil)
	if err != nil {
		return output, fmt.Errorf("%q: %v %v", name, sch, err)
	}
	_, _, err = resourceapply.ApplyDeployment(a.context,
		a.kubeClient.AppsV1(),
		recorder,
		deployment.(*appsv1.Deployment), 0)
	if err != nil {
		return output, fmt.Errorf("%q (%T): %v", name, deployment, err)
	}
	return output, nil
}

//ApplyDirectly applies standard kubernetes resources.
func (a Applier) ApplyDirectly(
	reader asset.ScenarioReader,
	values interface{},
	dryRun bool,
	headerFile string,
	files ...string) ([]string, error) {
	if dryRun {
		return a.MustTemplateAssets(reader, values, headerFile, files...)
	}
	var err error
	var memFSReader asset.ScenarioReader
	memFSReader, files, err = getFiles(reader, files, headerFile)
	if err != nil {
		return nil, err
	}
	// Sort all the files depending on their kind type
	files, err = a.Sort(memFSReader, values, headerFile, files...)
	if err != nil {
		return nil, err
	}
	recorder := events.NewInMemoryRecorder(helpers.GetExampleHeader())
	output := make([]string, 0)
	//Apply resources
	clients := resourceapply.NewClientHolder().
		WithAPIExtensionsClient(a.apiExtensionsClient).
		WithDynamicClient(a.dynamicClient).
		WithKubernetes(a.kubeClient)
	// Remove header files from the files as it should not be processed.
	files = asset.Delete(files, headerFile)
	resourceResults := resourceapply.
		ApplyDirectly(a.context, clients, recorder, a.cache, func(name string) ([]byte, error) {
			out, err := a.MustTemplateAsset(memFSReader, values, headerFile, name)
			if err != nil {
				return nil, err
			}
			output = append(output, string(out))
			return out, nil
		}, files...)
	//Check errors
	for _, result := range resourceResults {
		if result.Error != nil && !helpers.IsEmptyAsset(result.Error) {
			return output, fmt.Errorf("%q (%T): %v", result.File, result.Type, result.Error)
		}
	}
	return output, nil
}

func getFiles(reader asset.ScenarioReader, files []string, headerFile string) (asset.ScenarioReader, []string, error) {
	// Get all assets in the files array. The files could be a file name or directory
	files, err := reader.AssetNames(files, []string{}, headerFile)
	if err != nil {
		return nil, nil, err
	}
	// Files can contain multiple assets then we need to split all files
	// and we stored them in memory
	memFSReader, err := helpers.SplitFiles(reader, files)
	if err != nil {
		return nil, nil, err
	}

	// Get all assets in the files array. The files could be a file name or directory
	files, err = memFSReader.AssetNames(files, nil, headerFile)
	if err != nil {
		return nil, nil, err
	}
	return memFSReader, files, nil
}

//ApplyCustomResources applies custom resources
func (a Applier) ApplyCustomResources(
	reader asset.ScenarioReader,
	values interface{},
	dryRun bool,
	headerFile string,
	files ...string) ([]string, error) {
	output := make([]string, 0)
	// Remove header files from the files as it should not be processed.
	files = asset.Delete(files, headerFile)
	for _, name := range files {
		if name == headerFile {
			continue
		}
		asset, err := a.ApplyCustomResource(reader, values, dryRun, headerFile, name)
		if err != nil {
			if helpers.IsEmptyAsset(err) {
				continue
			}
			return output, err
		}
		output = append(output, string(asset))
	}
	return output, nil
}

//ApplyCustomResource applies a custom resource
func (a Applier) ApplyCustomResource(
	reader asset.ScenarioReader,
	values interface{},
	dryRun bool,
	headerFile string,
	name string) (string, error) {
	var output string
	if a.kubeClient == nil {
		return output, fmt.Errorf("missing apiExtensionsClient")
	}
	if a.dynamicClient == nil {
		return output, fmt.Errorf("missing dynamicClient")
	}
	asset, err := a.MustTemplateAsset(reader, values, headerFile, name)
	output = string(asset)
	if err != nil {
		return output, err
	}
	if dryRun {
		return output, nil
	}
	required, err := bytesToUnstructured(reader, asset)
	if err != nil {
		return output, err
	}
	gvks, _, err := genericScheme.ObjectKinds(required)
	if err != nil {
		return output, err
	}
	gvk := gvks[0]

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(a.kubeClient.Discovery()))
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return output, err
	}
	dr := a.dynamicClient.Resource(mapping.Resource)
	existing, err := dr.Namespace(required.GetNamespace()).Get(a.context, required.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			required := required.DeepCopy()
			actual, err := dr.Namespace(required.GetNamespace()).
				Create(a.context, required, metav1.CreateOptions{})
			a.GetCache().UpdateCachedResourceMetadata(required, actual)
			return output, err
		}
		return output, err
	}

	if a.GetCache().SafeToSkipApply(required, existing) {
		return output, nil
	}

	required.SetResourceVersion(existing.GetResourceVersion())
	actual, err := dr.Namespace(required.GetNamespace()).
		Update(a.context, required, metav1.UpdateOptions{})
	a.GetCache().UpdateCachedResourceMetadata(required, actual)

	if err != nil {
		return output, err
	}
	return output, nil
}

//bytesToUnstructured converts an asset to unstructured.
func bytesToUnstructured(reader asset.ScenarioReader, assetContent []byte) (*unstructured.Unstructured, error) {
	j, err := asset.ToJSON(assetContent)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(j, nil, u)
	if err != nil {
		//In case it is not a kube yaml
		if !runtime.IsMissingKind(err) {
			return nil, err
		}
	}
	return u, nil
}

//getTemplate generate the template for rendering.
func getTemplate(templateName string, customFuncMap template.FuncMap) *template.Template {
	tmpl := template.New(templateName).
		Option("missingkey=zero").
		Funcs(FuncMap())
	tmpl = tmpl.Funcs(TemplateFuncMap(tmpl)).
		Funcs(sprig.TxtFuncMap())
	if customFuncMap != nil {
		tmpl = tmpl.Funcs(customFuncMap)
	}
	return tmpl
}

//MustTemplateAssets render list of files
func (a Applier) MustTemplateAssets(reader asset.ScenarioReader,
	values interface{},
	headerFile string,
	files ...string) ([]string, error) {
	var err error
	var memFSReader asset.ScenarioReader
	memFSReader, files, err = getFiles(reader, files, headerFile)
	if err != nil {
		return nil, err
	}

	output := make([]string, 0)
	if a.kindOrder != nil {
		// Sort all the files depending on their kind type
		files, err = a.Sort(memFSReader, values, headerFile, files...)
		if err != nil {
			return output, err
		}
	}
	// Remove header files from the files as it should not be processed.
	files = asset.Delete(files, headerFile)
	for _, name := range files {
		if name == headerFile {
			continue
		}
		deploymentBytes, err := a.MustTemplateAsset(memFSReader, values, headerFile, name)
		if err != nil {
			if helpers.IsEmptyAsset(err) {
				continue
			}
			return output, err
		}
		output = append(output, string(deploymentBytes))
	}
	return output, nil
}

// MustTemplateAsset generates textual output for a template file name.
// The headerfile will be added to each file.
// Usually it contains nested template definitions as described
// https://golang.org/pkg/text/template/#hdr-Nested_template_definitions
// This allows to add functions which can be use in each file.
// The values object will be used to render the template
func (a Applier) MustTemplateAsset(reader asset.ScenarioReader,
	values interface{},
	headerFile, name string) ([]byte, error) {
	tmpl := getTemplate(name, a.templateFuncMap)
	h := []byte{}
	var err error
	if headerFile != "" {
		h, err = reader.Asset(headerFile)
		if err != nil {
			return nil, err
		}
	}
	hasMultipleAssets, err := helpers.HasMultipleAssets(reader, name)
	if err != nil {
		return nil, err
	}
	if hasMultipleAssets {
		memFSReader, files, err := getFiles(reader, []string{name}, headerFile)
		if err != nil {
			return nil, err
		}

		rendered, err := a.MustTemplateAssets(memFSReader, values, headerFile, files...)
		if err != nil {
			return nil, err
		}
		out := make([]byte, 0)
		for k, r := range rendered {
			out = append(out, []byte(r)...)
			if k == len(rendered)-1 {
				break
			}
			out = append(out, []byte("\n---\n")...)
		}
		return []byte(out), nil
	}
	b, err := reader.Asset(name)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	tmplParsed, err := tmpl.Parse(string(h))
	if err != nil {
		return nil, err
	}
	tmplParsed, err = tmplParsed.Parse(string(b))
	if err != nil {
		return nil, err
	}

	err = tmplParsed.Execute(&buf, values)
	if err != nil {
		return nil, err
	}

	//If the content is empty after rendering then returns an ErrorEmptyAssetAfterTemplating error.
	if helpers.IsEmpty(buf.Bytes()) {
		return nil, fmt.Errorf("asset %s becomes %s", name, helpers.ErrorEmptyAssetAfterTemplating)
	}

	if a.owner != nil {
		unstructuredObj := &unstructured.Unstructured{}
		j, err := yaml.YAMLToJSON(buf.Bytes())
		if err != nil {
			return nil, err
		}

		err = unstructuredObj.UnmarshalJSON(j)
		if err != nil {
			return nil, err
		}

		unstructuredObjOwnerRef := unstructuredObj.GetOwnerReferences()
		ownerRef, err := a.generateOwnerRef()
		if err != nil {
			return nil, err
		}

		var modified bool
		resourcemerge.MergeOwnerRefs(&modified,
			&unstructuredObjOwnerRef,
			[]metav1.OwnerReference{ownerRef})

		if modified {
			unstructuredObj.SetOwnerReferences(unstructuredObjOwnerRef)
			j, err := unstructuredObj.MarshalJSON()
			if err != nil {
				return nil, err
			}
			y, err := yaml.JSONToYAML(j)
			if err != nil {
				return nil, err
			}
			buf = *bytes.NewBuffer(y)
		}
	}
	return buf.Bytes(), nil
}

func (a Applier) generateOwnerRef() (ownerRef metav1.OwnerReference, err error) {
	err = addTypeInformationToObject(a.owner, a.scheme)
	if err != nil {
		return ownerRef, err
	}
	objectKind := a.owner.GetObjectKind()
	apiVersion, kind := objectKind.GroupVersionKind().ToAPIVersionAndKind()
	metaAccessor := meta.NewAccessor()
	ownerRef.APIVersion = apiVersion
	ownerRef.Kind = kind
	ownerRef.Name, err = metaAccessor.Name(a.owner)
	if err != nil {
		return ownerRef, err
	}
	ownerRef.UID, err = metaAccessor.UID(a.owner)
	if err != nil {
		return ownerRef, err
	}
	if *a.controller {
		ownerRef.Controller = a.controller
	}
	if *a.blockOwnerDeletion {
		ownerRef.BlockOwnerDeletion = a.blockOwnerDeletion
	}
	return ownerRef, err
}

func addTypeInformationToObject(obj runtime.Object, scheme *runtime.Scheme) error {
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return fmt.Errorf("missing apiVersion or kind and cannot assign it; %w", err)
	}

	for _, gvk := range gvks {
		if len(gvk.Kind) == 0 {
			continue
		}
		if len(gvk.Version) == 0 || gvk.Version == runtime.APIVersionInternal {
			continue
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
		break
	}

	return nil
}

func WriteOutput(fileName string, output []string) (err error) {
	if fileName == "" {
		return nil
	}
	var f *os.File
	if fileName == os.Stdout.Name() {
		f = os.Stdout
	} else {
		var err error
		f, err = os.Create(filepath.Clean(fileName))
		if err != nil {
			return err
		}
	}
	// defer f.Close()
	for _, s := range output {
		_, err := f.WriteString(fmt.Sprintf("%s\n---\n", s))
		if err != nil {
			if errClose := f.Close(); errClose != nil {
				return fmt.Errorf("failed to close %v after err %v on writing", errClose, err)
			}
			return err
		}
	}
	err = f.Close()
	return err
}
