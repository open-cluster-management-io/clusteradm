// Copyright Contributors to the Open Cluster Management project

package applierscenarios

import (
	"embed"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-cluster-management/applier/pkg/templateprocessor"

	"github.com/ghodss/yaml"
)

type ApplierScenarioReader interface {
	templateprocessor.TemplateReader
	ExtractAssets(prefix, dir string) error
}

type ApplierScenarioResourcesReader struct {
	files *embed.FS
}

var _ ApplierScenarioReader = &ApplierScenarioResourcesReader{
	files: nil,
}

func NewApplierScenarioResourcesReader(files *embed.FS) *ApplierScenarioResourcesReader {
	return &ApplierScenarioResourcesReader{
		files: files,
	}
}

func (r *ApplierScenarioResourcesReader) Asset(name string) ([]byte, error) {
	return r.files.ReadFile(name)
}

func (b *ApplierScenarioResourcesReader) AssetNames() ([]string, error) {
	assetNames := make([]string, 0)
	got, err := b.assetWalk(".")
	if err != nil {
		return nil, err
	}
	return append(assetNames, got...), nil
}

func (r *ApplierScenarioResourcesReader) assetWalk(f string) ([]string, error) {
	assets := make([]string, 0)
	file, err := r.files.Open(f)
	if err != nil {
		return assets, err
	}
	fs, err := file.Stat()
	if err != nil {
		return assets, err
	}
	if fs.IsDir() {
		de, err := r.files.ReadDir(f)
		if err != nil {
			return assets, err
		}
		for _, d := range de {
			di, err := d.Info()
			if err != nil {
				return assets, nil
			}
			assetsDir, err := r.assetWalk(filepath.Join(f, di.Name()))
			if err != nil {
				return assets, err
			}
			assets = append(assets, assetsDir...)
		}
		return assets, nil
	}
	return append(assets, f), nil
}

func (r *ApplierScenarioResourcesReader) ToJSON(b []byte) ([]byte, error) {
	return yaml.YAMLToJSON(b)
}

func (r *ApplierScenarioResourcesReader) ExtractAssets(prefix, dir string) error {
	assetNames, err := r.AssetNames()
	if err != nil {
		return err
	}
	for _, assetName := range assetNames {
		if !strings.HasPrefix(assetName, prefix) {
			continue
		}
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
