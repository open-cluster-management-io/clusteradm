// Copyright Contributors to the Open Cluster Management project
package helpers

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	operatorapiv1 "open-cluster-management.io/api/operator/v1"

	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
)

var (
	genericScheme = runtime.NewScheme()
	genericCodecs = serializer.NewCodecFactory(genericScheme)
	genericCodec  = genericCodecs.UniversalDeserializer()
)

func ApplyDeployment(
	client kubernetes.Interface,
	generationStatuses []operatorapiv1.GenerationStatus,
	reader ScenarioReader,
	templateName string,
	values map[string]interface{},
	name string) (operatorapiv1.GenerationStatus, error) {
	// s := scheme.Scheme
	// s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	recorder := events.NewInMemoryRecorder(GetExampleHeader())
	deploymentBytes, err := mustTempalteAsset(name, templateName, reader, values)
	if err != nil {
		return operatorapiv1.GenerationStatus{}, err
	}
	deployment, _, err := genericCodec.Decode(deploymentBytes, nil, nil)
	if err != nil {
		return operatorapiv1.GenerationStatus{}, fmt.Errorf("%q: %v", name, err)
	}
	generationStatus := NewGenerationStatus(appsv1.SchemeGroupVersion.WithResource("deployments"), deployment)
	currentGenerationStatus := FindGenerationStatus(generationStatuses, generationStatus)

	if currentGenerationStatus != nil {
		generationStatus.LastGeneration = currentGenerationStatus.LastGeneration
	}
	updatedDeployment, updated, err := resourceapply.ApplyDeployment(
		client.AppsV1(),
		recorder,
		deployment.(*appsv1.Deployment), generationStatus.LastGeneration)
	if err != nil {
		return generationStatus, fmt.Errorf("%q (%T): %v", name, deployment, err)
	}

	if updated {
		generationStatus.LastGeneration = updatedDeployment.ObjectMeta.Generation
	}

	return generationStatus, nil
}

func NewGenerationStatus(gvr schema.GroupVersionResource, object runtime.Object) operatorapiv1.GenerationStatus {
	accessor, _ := meta.Accessor(object)
	return operatorapiv1.GenerationStatus{
		Group:          gvr.Group,
		Version:        gvr.Version,
		Resource:       gvr.Resource,
		Namespace:      accessor.GetNamespace(),
		Name:           accessor.GetName(),
		LastGeneration: accessor.GetGeneration(),
	}
}

func FindGenerationStatus(generationStatuses []operatorapiv1.GenerationStatus, generation operatorapiv1.GenerationStatus) *operatorapiv1.GenerationStatus {
	for i := range generationStatuses {
		if generationStatuses[i].Group != generation.Group {
			continue
		}
		if generationStatuses[i].Resource != generation.Resource {
			continue
		}
		if generationStatuses[i].Version != generation.Version {
			continue
		}
		if generationStatuses[i].Name != generation.Name {
			continue
		}
		if generationStatuses[i].Namespace != generation.Namespace {
			continue
		}
		return &generationStatuses[i]
	}
	return nil
}

func ApplyDirectly(clients *resourceapply.ClientHolder,
	reader ScenarioReader,
	templateName string,
	values map[string]interface{},
	files ...string) []resourceapply.ApplyResult {
	recorder := events.NewInMemoryRecorder(GetExampleHeader())
	return resourceapply.ApplyDirectly(clients, recorder, func(name string) ([]byte, error) {
		return mustTempalteAsset(name, templateName, reader, values)
	}, files...)
}

func getTemplate(templateName string) *template.Template {
	tmpl := template.New(templateName).
		Option("missingkey=zero").
		Funcs(FuncMap())
	tmpl = tmpl.Funcs(TemplateFuncMap(tmpl)).
		Funcs(sprig.TxtFuncMap())
	return tmpl
}

func mustTempalteAsset(file, templateName string, reader ScenarioReader, values map[string]interface{}) ([]byte, error) {
	tmpl := getTemplate(templateName)
	b, err := reader.Asset(file)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	tmplParsed, err := tmpl.Parse(string(b))
	if err != nil {
		return nil, err
	}

	err = tmplParsed.Execute(&buf, values)
	if err != nil {
		return nil, err
	}

	// recorder.Eventf("templated:\n%s\n---", buf.String())
	trim := strings.TrimSuffix(buf.String(), "\n")
	trim = strings.TrimSpace(trim)
	if len(trim) == 0 {
		return nil, nil
	}
	return buf.Bytes(), nil
}
