// Copyright Contributors to the Open Cluster Management project
package helpers

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog"

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
	reader ScenarioReader,
	templateName string,
	values interface{},
	files ...string) error {
	genericScheme.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	recorder := events.NewInMemoryRecorder(GetExampleHeader())
	for _, name := range files {
		deploymentBytes, err := mustTempalteAsset(name, templateName, reader, values)
		if err != nil {
			return err
		}
		deployment, sch, err := genericCodec.Decode(deploymentBytes, nil, nil)
		if err != nil {
			return fmt.Errorf("%q: %v %v", name, sch, err)
		}
		_, _, err = resourceapply.ApplyDeployment(
			client.AppsV1(),
			recorder,
			deployment.(*appsv1.Deployment), 0)
		if err != nil {
			return fmt.Errorf("%q (%T): %v", name, deployment, err)
		}
	}
	return nil
}

func ApplyDirectly(clients *resourceapply.ClientHolder,
	reader ScenarioReader,
	templateName string,
	values interface{},
	files ...string) error {
	recorder := events.NewInMemoryRecorder(GetExampleHeader())
	resourceResults := resourceapply.ApplyDirectly(clients, recorder, func(name string) ([]byte, error) {
		return mustTempalteAsset(name, templateName, reader, values)
	}, files...)
	for _, result := range resourceResults {
		if result.Error != nil {
			return fmt.Errorf("%q (%T): %v", result.File, result.Type, result.Error)
		}
	}
	return nil
}

func ApplyCustomResouces(client dynamic.Interface,
	discoveryClient discovery.DiscoveryInterface,
	reader ScenarioReader,
	templateName string,
	values interface{},
	files ...string) error {
	for _, name := range files {
		asset, err := mustTempalteAsset(name, templateName, reader, values)
		if err != nil {
			return err
		}
		u, err := bytesToUnstructured(reader, asset)
		if err != nil {
			return err
		}
		gvks, _, err := genericScheme.ObjectKinds(u)
		if err != nil {
			return err
		}
		gvk := gvks[0]
		mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}
		// resource, err := getResource(u.GetKind())
		// if err != nil {
		// 	return err
		// }
		// dr := client.Resource(gvk.GroupVersion().WithResource(resource))
		dr := client.Resource(mapping.Resource)
		ug, err := dr.Namespace(u.GetNamespace()).Get(context.TODO(), u.GetName(), metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				_, err = dr.Namespace(u.GetNamespace()).
					Create(context.TODO(), u, metav1.CreateOptions{})
			}
		} else {
			u.SetResourceVersion(ug.GetResourceVersion())
			_, err = dr.Namespace(u.GetNamespace()).
				Update(context.TODO(), u, metav1.UpdateOptions{})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func getResource(kind string) (string, error) {
	switch kind {
	case "ClusterManager":
		return "clustermanagers", nil
	case "Klusterlet":
		return "klusterlets", nil
	default:
		return "", fmt.Errorf("kind: %s not supported", kind)
	}
}

func bytesToUnstructured(reader ScenarioReader, asset []byte) (*unstructured.Unstructured, error) {
	j, err := reader.ToJSON(asset)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(j, nil, u)
	if err != nil {
		klog.V(5).Infof("Error: %s", err)
		//In case it is not a kube yaml
		if !runtime.IsMissingKind(err) {
			return nil, err
		}
	}
	return u, nil
}

func getTemplate(templateName string) *template.Template {
	tmpl := template.New(templateName).
		Option("missingkey=zero").
		Funcs(FuncMap())
	tmpl = tmpl.Funcs(TemplateFuncMap(tmpl)).
		Funcs(sprig.TxtFuncMap())
	return tmpl
}

func mustTempalteAsset(name, templateName string, reader ScenarioReader, values interface{}) ([]byte, error) {
	tmpl := getTemplate(templateName)
	b, err := reader.Asset(name)
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
