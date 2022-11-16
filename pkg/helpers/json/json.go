// Copyright Contributors to the Open Cluster Management project
package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type HubInfo struct {
	HubToken     string `json:"hub-token"`
	HubApiserver string `json:"hub-apiserver"`
}

func WriteJsonOutput(w io.Writer, val interface{}) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	err = json.Indent(&out, b, "", "  ")
	if err != nil {
		return err
	}
	_, err = out.WriteTo(w)
	if err != nil {
		return err
	}
	fmt.Fprintln(w)
	return nil
}
