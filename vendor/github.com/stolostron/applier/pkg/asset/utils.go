// Copyright Red Hat
package asset

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/klog/v2"
)

func ToJSON(b []byte) ([]byte, error) {
	j, err := yaml.YAMLToJSON(b)
	if err != nil {
		klog.Errorf("err:%s\nyaml:\n%s", err, string(b))
		return nil, err
	}
	return j, nil
}

func ExtractAssets(r ScenarioReader, prefix, dir string, excluded []string, headerFile string) error {
	assetNames, err := r.AssetNames([]string{prefix}, excluded, headerFile)
	if err != nil {
		return err
	}
	for _, assetName := range assetNames {
		relPath, err := filepath.Rel(prefix, assetName)
		if err != nil {
			return err
		}
		path := filepath.Join(dir, relPath)

		if relPath == "." {
			path = filepath.Join(dir, filepath.Base(assetName))
		}
		err = os.MkdirAll(filepath.Dir(path), os.FileMode(0700))
		if err != nil {
			return err
		}
		data, err := r.Asset(assetName)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(path, data, os.FileMode(0600))
		if err != nil {
			return err
		}
	}
	return nil
}

func isExcluded(f string, prefixes, excluded []string) bool {
	isExcluded := false
	for _, e := range excluded {
		if f == e {
			isExcluded = true
		}
	}
	// Already excluded
	if isExcluded {
		return true
	}
	// No extra test to do
	if prefixes == nil {
		return false
	}
	isExcluded = true
	for _, d := range prefixes {
		if strings.HasPrefix(f, d) {
			isExcluded = false
		}
	}
	return isExcluded
}

func AppendItNotExists(a []string, e string) []string {
	if len(e) == 0 {
		return a
	}
	exists := false
	for _, v := range a {
		if v == e {
			exists = true
		}
	}
	if !exists {
		a = append(a, e)
	}
	return a
}

func Delete(a []string, e string) []string {
	p := -1
	for i, v := range a {
		if v == e {
			p = i
		}
	}
	if p != -1 {
		a[p] = a[len(a)-1]
		return a[:len(a)-1]
	}
	return a
}
