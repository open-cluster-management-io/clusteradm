// Copyright Red Hat

package helpers

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/stolostron/applier/pkg/asset"
)

func HasMultipleAssets(reader asset.ScenarioReader, path string) (bool, error) {
	// Seems ^ and $ are not working
	re := regexp.MustCompile("\n-{3}[\\s]*\n")
	b, err := reader.Asset(path)
	if err != nil {
		return false, err
	}
	s := string(b)
	splitted := re.Split(s, -1)
	return len(splitted) != 1, nil
}

func SplitFiles(reader asset.ScenarioReader, paths []string) (*asset.MemFS, error) {
	memFs := asset.NewMemFSReader()
	re := regexp.MustCompile("\n-{3}[\\s]*\n")

	for _, p := range paths {
		b, err := reader.Asset(p)
		if err != nil {
			return nil, err
		}
		splitted := re.Split(string(b), -1)

		cleanedSplitted := make([]string, 0)

		for _, s := range splitted {
			if !IsEmpty([]byte(s)) {
				cleanedSplitted = append(cleanedSplitted, s)
			}
		}

		if len(cleanedSplitted) == 1 {
			memFs.AddAsset(p, []byte(cleanedSplitted[0]))
			continue
		}
		for k, s := range cleanedSplitted {
			memFs.AddAsset(filepath.Join(filepath.Dir(p), fmt.Sprintf("%s.%d%s", filepath.Base(p), k, filepath.Ext(p))), []byte(s))
		}
	}
	return memFs, nil
}
