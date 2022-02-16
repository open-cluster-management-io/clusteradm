// Copyright Contributors to the Open Cluster Management project
package version

import (
	"errors"
)

type VersionBundle struct {	
	Registration string
	Placement string
	Work string
	Operator string
}	


func GetVersionBundle(version string ) (VersionBundle ,error){

	versionBundleList := map[string]VersionBundle{}

	// default 
	versionBundleList["default"] = VersionBundle{
		Registration: "latest",
		Placement: "latest",
		Work: "latest",
		Operator: "latest",
	}
	
	// predifined bundle version 
	versionBundleList["0.5.0"] = VersionBundle{
		Registration: "0.5.0",
		Placement: "0.2.0",
		Work: "0.5.0",
		Operator: "0.5.0",

	}

	versionBundleList["0.6.0"] = VersionBundle{
		Registration: "0.6.0",
		Placement: "0.3.0",
		Work: "0.6.0",
		Operator: "0.6.0",
	}


	if val, ok := versionBundleList[version]; ok {
		return val , nil
	} 
	no_version_error := errors.New("couldn't find the requested version bundle")
	return  VersionBundle{}, no_version_error
}

