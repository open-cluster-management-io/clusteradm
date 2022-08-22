// Copyright Red Hat
package asset

type ScenarioReader interface {
	// Retrieve an asset from the data source
	Asset(templatePath string) ([]byte, error)
	// List all available assets in the data source
	// with a prefix of one of the name in the files array
	// if the prefix array is nil, the prefix will not be tested
	// excluding the name in the excluded array
	AssetNames(prefixes, excluded []string, headerFile string) ([]string, error)
}
