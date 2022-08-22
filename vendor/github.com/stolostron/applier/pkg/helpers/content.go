// Copyright Red Hat

package helpers

import (
	"regexp"
	"strings"
)

const (
	ErrorEmptyAssetAfterTemplating = "ERROR_EMPTY_ASSET_AFTER_TEMPLATING"
)

//IsEmpty check if a content is empty after removing comments and blank lines.
func IsEmpty(body []byte) bool {
	//Remove comments
	reComment := regexp.MustCompile("#.*")
	bodyNoComment := reComment.ReplaceAll(body, nil)
	//Remove ---
	reSeparator := regexp.MustCompile("^---")
	bodyNoSeparator := reSeparator.ReplaceAll(bodyNoComment, nil)
	//Remove blank lines
	trim := strings.TrimSuffix(string(bodyNoSeparator), "\n")
	trim = strings.TrimSpace(trim)

	return len(trim) == 0
}

//IsEmptyAsset returns true if the error is ErrorEmptyAssetAfterTemplating
func IsEmptyAsset(err error) bool {
	return strings.Contains(err.Error(), ErrorEmptyAssetAfterTemplating)
}
