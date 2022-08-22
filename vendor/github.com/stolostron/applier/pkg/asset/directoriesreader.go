// Copyright Red Hat

package asset

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

//YamlFileReader defines a reader for yaml files
type YamlFileReader struct {
	header string
	paths  []string
	files  []string
}

var _ ScenarioReader = &YamlFileReader{
	header: "",
	paths:  []string{},
	files:  []string{},
}

//NewDirectoriesReader constructs a new YamlFileReader
func NewDirectoriesReader(
	header string,
	paths []string,
) (*YamlFileReader, error) {
	reader := &YamlFileReader{
		header: header,
		paths:  paths,
	}
	files, err := reader.AssetNames(paths, nil, header)
	if err != nil {
		return nil, err
	}
	reader.files = files

	return reader, nil
}

//Asset returns an asset
func (r *YamlFileReader) Asset(
	name string,
) ([]byte, error) {
	found := false
	for _, p := range r.files {
		if p == name {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("file %s is not part of the assets", name)
	}
	return ioutil.ReadFile(filepath.Clean(name))
}

//AssetNames returns the name of all assets
func (r *YamlFileReader) AssetNames(prefixes, excluded []string, headerFile string) ([]string, error) {
	assetNames := make([]string, 0)
	visit := func(path string, fileInfo os.FileInfo, err error) error {
		if fileInfo == nil {
			return fmt.Errorf("paths %s doesn't exist", path)
		}
		if fileInfo.IsDir() {
			return nil
		}
		if isExcluded(path, []string{path}, excluded) {
			return nil
		}
		assetNames = append(assetNames, path)
		return nil
	}

	for _, p := range r.paths {
		if err := filepath.Walk(p, visit); err != nil {
			return assetNames, err
		}
	}
	// The header file must be added in the assetNames as it is retrieved latter
	// to render asset in the MustTemplateAsset
	assetNames = AppendItNotExists(assetNames, headerFile)
	return assetNames, nil
}
