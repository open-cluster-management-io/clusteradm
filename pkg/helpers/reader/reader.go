// Copyright Contributors to the Open Cluster Management project

package reader

import (
	"bytes"
	"embed"
	"fmt"
	"strings"

	"github.com/jonboulle/clockwork"
	"github.com/openshift/library-go/pkg/assets"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/apply"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	kubectlutil "k8s.io/kubectl/pkg/util"
)

const yamlSeparator = "\n---\n"

type ResourceReader struct {
	builder *resource.Builder
	dryRun  bool
	streams genericiooptions.IOStreams
	raw     []byte
	f       cmdutil.Factory
}

func NewResourceReader(f cmdutil.Factory, dryRun bool, streams genericiooptions.IOStreams) *ResourceReader {
	return &ResourceReader{
		builder: f.NewBuilder().Unstructured(),
		dryRun:  dryRun,
		streams: streams,
		raw:     []byte{},
		f:       f,
	}
}

func (r *ResourceReader) RawAppliedResources() []byte {
	return r.raw
}

func (r *ResourceReader) ApplyRaw(raw [][]byte) error {
	var errs []error
	for _, objData := range raw {
		if err := r.apply(objData); err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

func (r *ResourceReader) apply(raw []byte) error {
	rb := r.builder.
		Stream(bytes.NewReader(raw), "local").
		Flatten().
		Do()
	if r.dryRun {
		// Ignore CRD not installed error when dry-run
		rb.IgnoreErrors(func(err error) bool {
			return err != nil && strings.Contains(err.Error(), "ensure CRDs are installed first")
		})
	}
	infos, err := rb.Infos()
	if err != nil {
		return err
	}

	var errs []error
	for _, object := range infos {
		if err := r.applyOneObject(object); err != nil {
			errs = append(errs, err)
		}
	}

	if r.dryRun {
		fmt.Fprintf(r.streams.Out, "%s", string(raw))
	}
	r.raw = append(r.raw, raw...)
	return utilerrors.NewAggregate(errs)
}

func (r *ResourceReader) Apply(fs embed.FS, config interface{}, files ...string) error {
	rawObjects := []byte{}
	for _, file := range files {
		template, err := fs.ReadFile(file)
		if err != nil {
			return err
		}
		objData := assets.MustCreateAssetFromTemplate(file, template, config).Data
		rawObjects = append(rawObjects, objData...)
		rawObjects = append(rawObjects, []byte(yamlSeparator)...)
	}

	return r.apply(rawObjects)
}

func (r *ResourceReader) applyOneObject(info *resource.Info) error {
	if len(info.Name) == 0 {
		metadata, _ := meta.Accessor(info.Object)
		generatedName := metadata.GetGenerateName()
		if len(generatedName) > 0 {
			return fmt.Errorf("from %s: cannot use generate name with apply", generatedName)
		}
	}

	helper := resource.NewHelper(info.Client, info.Mapping).
		DryRun(r.dryRun)

	modified, err := kubectlutil.GetModifiedConfiguration(info.Object, false, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return cmdutil.AddSourceToErr(fmt.Sprintf("retrieving modified configuration from:\n%s\nfor:", info.String()), info.Source, err)
	}

	if err := info.Get(); err != nil {
		if !errors.IsNotFound(err) {
			return cmdutil.AddSourceToErr(fmt.Sprintf("retrieving current configuration of:\n%s\nfrom server for:", info.String()), info.Source, err)
		}

		if !r.dryRun {
			// Then create the resource and skip the three-way merge
			obj, err := helper.Create(info.Namespace, true, info.Object)
			if err != nil {
				return cmdutil.AddSourceToErr("creating", info.Source, err)
			}
			if err := info.Refresh(obj, true); err != nil {
				return err
			}
		}
	}

	if !r.dryRun {
		patcher := newPatcher(info, helper, r.f)
		patchBytes, patchedObject, err := patcher.Patch(info.Object, modified, info.Source, info.Namespace, info.Name, r.streams.ErrOut)
		if err != nil {
			return cmdutil.AddSourceToErr(fmt.Sprintf("applying patch:\n%s\nto:\n%v\nfor:", patchBytes, info), info.Source, err)
		}

		if err := info.Refresh(patchedObject, true); err != nil {
			return err
		}
	}

	return nil
}

func (r *ResourceReader) Delete(fs embed.FS, config interface{}, files ...string) error {
	rawObjects := []byte{}
	for _, file := range files {
		template, err := fs.ReadFile(file)
		if err != nil {
			return err
		}
		objData := assets.MustCreateAssetFromTemplate(file, template, config).Data
		rawObjects = append(rawObjects, objData...)
		rawObjects = append(rawObjects, []byte(yamlSeparator)...)
	}

	rb := r.builder.
		Stream(bytes.NewReader(rawObjects), "local").
		Flatten().
		Do()
	infos, err := rb.Infos()
	if err != nil {
		return err
	}

	var errs []error
	for _, object := range infos {
		if err := r.deleteOneObject(object); err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func (r *ResourceReader) deleteOneObject(info *resource.Info) error {
	helper := resource.NewHelper(info.Client, info.Mapping).
		DryRun(r.dryRun)
	_, err := helper.Delete(info.Namespace, info.Name)
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func newPatcher(info *resource.Info, helper *resource.Helper, f cmdutil.Factory) *apply.Patcher {
	return &apply.Patcher{
		Mapping:       info.Mapping,
		Helper:        helper,
		Overwrite:     true,
		BackOff:       clockwork.NewRealClock(),
		OpenAPIGetter: f,
	}
}
