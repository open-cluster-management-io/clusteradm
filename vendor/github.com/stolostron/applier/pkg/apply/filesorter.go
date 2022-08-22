// Copyright Red Hat
package apply

import (
	"sort"

	"github.com/stolostron/applier/pkg/asset"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//KindsOrder ...
type KindsOrder []string

//DefaultKindsOrder the default order
var DefaultCreateUpdateKindsOrder KindsOrder = []string{
	"Namespace",
	"NetworkPolicy",
	"ResourceQuota",
	"LimitRange",
	"PodSecurityPolicy",
	"PodDisruptionBudget",
	"ServiceAccount",
	"Secret",
	"SecretList",
	"ConfigMap",
	"StorageClass",
	"PersistentVolume",
	"PersistentVolumeClaim",
	"CustomResourceDefinition",
	"ClusterRole",
	"ClusterRoleList",
	"ClusterRoleBinding",
	"ClusterRoleBindingList",
	"Role",
	"RoleList",
	"RoleBinding",
	"RoleBindingList",
	"Service",
	"DaemonSet",
	"Pod",
	"ReplicationController",
	"ReplicaSet",
	"Deployment",
	"HorizontalPodAutoscaler",
	"StatefulSet",
	"Job",
	"CronJob",
	"Ingress",
	"APIService",
}

var NoCreateUpdateKindsOrder KindsOrder = []string{}

type FileInfo struct {
	FileName   string
	Kind       string
	Name       string
	Namespace  string
	APIVersion string
}

func (a *Applier) Sort(reader asset.ScenarioReader,
	values interface{},
	headerFile string,
	files ...string) ([]string, error) {
	// If no kind order
	if len(a.kindOrder) == 0 {
		return files, nil
	}

	filesInfo, err := a.GetFileInfo(reader, values, headerFile, files...)
	if err != nil {
		return nil, err
	}
	a.sortFiles(filesInfo)

	files = make([]string, len(filesInfo))
	for i, fileInfo := range filesInfo {
		files[i] = fileInfo.FileName
	}
	return files, nil
}

func (a *Applier) GetFileInfo(reader asset.ScenarioReader,
	values interface{},
	headerFile string,
	files ...string) ([]FileInfo, error) {
	filesInfo := make([]FileInfo, 0)
	// Remove header files from the files as it should not be processed.
	files = asset.Delete(files, headerFile)
	for _, name := range files {
		b, err := a.MustTemplateAsset(reader, values, headerFile, name)
		if err != nil {
			return nil, err
		}
		unstructuredObj := &unstructured.Unstructured{}
		j, err := asset.ToJSON(b)
		if err != nil {
			return nil, err
		}

		err = unstructuredObj.UnmarshalJSON(j)
		if err != nil {
			return nil, err
		}
		filesInfo = append(filesInfo,
			FileInfo{
				FileName:   name,
				Kind:       unstructuredObj.GetKind(),
				Name:       unstructuredObj.GetName(),
				Namespace:  unstructuredObj.GetNamespace(),
				APIVersion: unstructuredObj.GetAPIVersion(),
			})
	}
	return filesInfo, nil
}

//sortUnstructuredForApply sorts a list on unstructured
func (a *Applier) sortFiles(filesInfo []FileInfo) {
	sort.Slice(filesInfo[:], func(i, j int) bool {
		return a.less(filesInfo[i], filesInfo[j])
	})
}

func (a *Applier) less(fileInfo1, fileInfo2 FileInfo) bool {
	if a.weight(fileInfo1) == a.weight(fileInfo2) {
		if fileInfo1.Namespace == fileInfo2.Namespace {
			return fileInfo1.Name < fileInfo2.Name
		}
		return fileInfo1.Namespace < fileInfo2.Namespace
	}
	return a.weight(fileInfo1) < a.weight(fileInfo2)
}

func (a *Applier) weight(fileInfo FileInfo) int {
	defaultWeight := len(a.kindOrder)
	for i, k := range a.kindOrder {
		if k == fileInfo.Kind {
			return i
		}
	}
	return defaultWeight
}
