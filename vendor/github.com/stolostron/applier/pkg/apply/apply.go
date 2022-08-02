// Copyright Contributors to the Open Cluster Management project
package apply

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/stolostron/applier/pkg/asset"
	"github.com/stolostron/applier/pkg/helpers"

	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"

	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
)

const (
	ErrorEmptyAssetAfterTemplating = "ERROR_EMPTY_ASSET_AFTER_TEMPLATING"
)

var (
	genericScheme = runtime.NewScheme()
	genericCodecs = serializer.NewCodecFactory(genericScheme)
	genericCodec  = genericCodecs.UniversalDeserializer()
)

func (a *Applier) GetCache() resourceapply.ResourceCache {
	return a.cache
}

//ApplyDeployments applies a appsv1.Deployment template
func (a *Applier) ApplyDeployments(
	reader asset.ScenarioReader,
	values interface{},
	dryRun bool,
	headerFile string,
	files ...string) ([]string, error) {
	output := make([]string, 0)
	//Render each file
	for _, name := range files {
		deployment, err := a.ApplyDeployment(reader, values, dryRun, headerFile, name)
		if err != nil {
			if IsEmptyAsset(err) {
				continue
			}
			return output, err
		}
		output = append(output, deployment)
	}
	return output, nil
}

//ApplyDeployment apply a deployment
func (a *Applier) ApplyDeployment(
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
func (a *Applier) ApplyDirectly(
	reader asset.ScenarioReader,
	values interface{},
	dryRun bool,
	headerFile string,
	files ...string) ([]string, error) {
	if dryRun {
		return a.MustTemplateAssets(reader, values, headerFile, files...)
	}
	recorder := events.NewInMemoryRecorder(helpers.GetExampleHeader())
	output := make([]string, 0)
	//Apply resources
	clients := resourceapply.NewClientHolder().
		WithAPIExtensionsClient(a.apiExtensionsClient).
		WithDynamicClient(a.dynamicClient).
		WithKubernetes(a.kubeClient)
	resourceResults := resourceapply.
		ApplyDirectly(a.context, clients, recorder, a.cache, func(name string) ([]byte, error) {
			out, err := a.MustTemplateAsset(reader, values, headerFile, name)
			if err != nil {
				return nil, err
			}
			output = append(output, string(out))
			return out, nil
		}, files...)
	//Check errors
	for _, result := range resourceResults {
		if result.Error != nil && !IsEmptyAsset(result.Error) {
			return output, fmt.Errorf("%q (%T): %v", result.File, result.Type, result.Error)
		}
	}
	return output, nil
}

//ApplyCustomResources applies custom resources
func (a *Applier) ApplyCustomResources(
	reader asset.ScenarioReader,
	values interface{},
	dryRun bool,
	headerFile string,
	files ...string) ([]string, error) {
	output := make([]string, 0)
	for _, name := range files {
		asset, err := a.ApplyCustomResource(reader, values, dryRun, headerFile, name)
		if err != nil {
			if IsEmptyAsset(err) {
				continue
			}
			return output, err
		}
		output = append(output, string(asset))
	}
	return output, nil
}

//ApplyCustomResource applies a custom resource
func (a *Applier) ApplyCustomResource(
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
func bytesToUnstructured(reader asset.ScenarioReader, asset []byte) (*unstructured.Unstructured, error) {
	j, err := reader.ToJSON(asset)
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
func (a *Applier) MustTemplateAssets(reader asset.ScenarioReader,
	values interface{},
	headerFile string,
	files ...string) ([]string, error) {
	output := make([]string, 0)
	for _, name := range files {
		if name == headerFile {
			continue
		}
		deploymentBytes, err := a.MustTemplateAsset(reader, values, headerFile, name)
		if err != nil {
			if IsEmptyAsset(err) {
				continue
			}
			return output, err
		}
		output = append(output, string(deploymentBytes))
	}
	return output, nil
}

//MustTemplateAsset generates textual output for a template file name.
//The headerfile will be added to each file.
//Usually it contains nested template definitions as described
// https://golang.org/pkg/text/template/#hdr-Nested_template_definitions
//This allows to add functions which can be use in each file.
//The values object will be used to render the template
func (a *Applier) MustTemplateAsset(reader asset.ScenarioReader,
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
	if isEmpty(buf.Bytes()) {
		return nil, fmt.Errorf("asset %s becomes %s", name, ErrorEmptyAssetAfterTemplating)
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

func (a *Applier) generateOwnerRef() (ownerRef metav1.OwnerReference, err error) {
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

//isEmpty check if a content is empty after removing comments and blank lines.
func isEmpty(body []byte) bool {
	//Remove comments
	re := regexp.MustCompile("#.*")
	bodyNoComment := re.ReplaceAll(body, nil)
	//Remove blank lines
	trim := strings.TrimSuffix(string(bodyNoComment), "\n")
	trim = strings.TrimSpace(trim)

	return len(trim) == 0
}

//IsEmptyAsset returns true if the error is ErrorEmptyAssetAfterTemplating
func IsEmptyAsset(err error) bool {
	return strings.Contains(err.Error(), ErrorEmptyAssetAfterTemplating)
}

func WriteOutput(fileName string, output []string) (err error) {
	if fileName == "" {
		return nil
	}
	f, err := os.Create(filepath.Clean(fileName))
	if err != nil {
		return err
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
