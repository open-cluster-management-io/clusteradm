// Copyright Contributors to the Open Cluster Management project
package bindata

import (
	"github.com/ghodss/yaml"
)

type Bindata struct{}

func (b *Bindata) Asset(name string) ([]byte, error) {
	return Asset(name)
}

func (b *Bindata) AssetNames() ([]string, error) {
	return AssetNames(), nil
}

func (*Bindata) ToJSON(b []byte) ([]byte, error) {
	return yaml.YAMLToJSON(b)
}

func NewBindataReader() *Bindata {
	return &Bindata{}
}
