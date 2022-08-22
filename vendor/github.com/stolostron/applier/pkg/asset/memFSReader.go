// Copyright Red Hat

package asset

type MemFS struct {
	files []string
	data  map[string][]byte
}

var _ ScenarioReader = &ScenarioResourcesReader{
	files: nil,
}

func NewMemFSReader() *MemFS {
	return &MemFS{
		// This array is used to keep the same order
		files: make([]string, 0),
		data:  make(map[string][]byte, 0),
	}
}

func (r *MemFS) AddAssetsFromScenarioReader(reader ScenarioReader, headerFile string) error {
	assets, err := reader.AssetNames(nil, nil, headerFile)
	if err != nil {
		return err
	}
	for _, asset := range assets {
		b, err := reader.Asset(asset)
		if err != nil {
			return err
		}
		r.AddAsset(asset, b)
	}
	return nil
}

func (r *MemFS) AddAsset(fileName string, data []byte) {
	r.files = append(r.files, fileName)
	r.data[fileName] = data
}

func (r *MemFS) Asset(name string) ([]byte, error) {
	return r.data[name], nil
}

func (r *MemFS) AssetNames(prefixes, excluded []string, headerFile string) ([]string, error) {
	assetNames := make([]string, 0)
	for _, f := range r.files {
		if !isExcluded(f, prefixes, excluded) {
			assetNames = append(assetNames, f)
		}
	}
	// The header file must be added in the assetNames as it is retrieved latter
	// to render asset in the MustTemplateAsset
	assetNames = AppendItNotExists(assetNames, headerFile)
	return assetNames, nil
}
