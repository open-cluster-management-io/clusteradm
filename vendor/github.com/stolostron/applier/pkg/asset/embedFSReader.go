// Copyright Red Hat

package asset

import (
	"embed"
	"path/filepath"
)

type ScenarioResourcesReader struct {
	files *embed.FS
}

var _ ScenarioReader = &ScenarioResourcesReader{
	files: nil,
}

func NewScenarioResourcesReader(files *embed.FS) *ScenarioResourcesReader {
	return &ScenarioResourcesReader{
		files: files,
	}
}

func (r *ScenarioResourcesReader) Asset(name string) ([]byte, error) {
	return r.files.ReadFile(name)
}

func (r *ScenarioResourcesReader) AssetNames(prefixes, excluded []string, headerFile string) ([]string, error) {
	assetNames := make([]string, 0)
	got, err := r.assetWalk(".")
	if err != nil {
		return nil, err
	}
	for _, f := range got {
		if !isExcluded(f, prefixes, excluded) {
			assetNames = append(assetNames, f)
		}
	}
	// The header file must be added in the assetNames as it is retrieved latter
	// to render asset in the MustTemplateAsset
	assetNames = AppendItNotExists(assetNames, headerFile)
	return assetNames, nil
}

func (r *ScenarioResourcesReader) assetWalk(f string) ([]string, error) {
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
