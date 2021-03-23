// Copyright Contributors to the Open Cluster Management project

package resources

import (
	"embed"
	"path/filepath"

	"github.com/ghodss/yaml"
)

type Resources struct{}

//Needed to scenarios/*/*/*/* to include the _helpers.tpl located in:
//scenarios/create/hub/common/_helpers.tpl and
//scenarios/destroy/hub/common/_helpers.tpl
//go:embed scenarios scenarios/*/*/*/_helpers.tpl
var files embed.FS

func (*Resources) Asset(name string) ([]byte, error) {
	return files.ReadFile(name)
}

func (b *Resources) AssetNames() ([]string, error) {
	assetNames := make([]string, 0)
	got, err := b.assetWalk(".")
	if err != nil {
		return nil, err
	}
	return append(assetNames, got...), nil
}

func (b *Resources) assetWalk(f string) ([]string, error) {
	assets := make([]string, 0)
	file, err := files.Open(f)
	if err != nil {
		return assets, err
	}
	fs, err := file.Stat()
	if err != nil {
		return assets, err
	}
	if fs.IsDir() {
		de, err := files.ReadDir(f)
		if err != nil {
			return assets, err
		}
		for _, d := range de {
			di, err := d.Info()
			if err != nil {
				return assets, nil
			}
			assetsDir, err := b.assetWalk(filepath.Join(f, di.Name()))
			if err != nil {
				return assets, err
			}
			assets = append(assets, assetsDir...)
		}
		return assets, nil
	}
	return append(assets, f), nil
}

func (*Resources) ToJSON(b []byte) ([]byte, error) {
	return yaml.YAMLToJSON(b)
}

func NewResourcesReader() *Resources {
	return &Resources{}
}
