// Copyright Contributors to the Open Cluster Management project
package create

import (
	"context"
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/utils/ptr"

	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
)

var _ = ginkgo.Describe("addon create", func() {

	var (
		suffix string
		err    error
	)

	ginkgo.BeforeEach(func() {
		suffix = rand.String(5)
	})

	streams := genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	ginkgo.Context("parsePlacementRef", func() {
		ginkgo.It("Should parse valid placement-ref correctly", func() {
			o := &Options{
				PlacementRef: "test-namespace/test-placement",
			}

			namespace, name, err := o.parsePlacementRef()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(namespace).To(gomega.Equal("test-namespace"))
			gomega.Expect(name).To(gomega.Equal("test-placement"))
		})

		ginkgo.It("Should handle empty placement-ref", func() {
			o := &Options{
				PlacementRef: "",
			}

			namespace, name, err := o.parsePlacementRef()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(namespace).To(gomega.Equal(""))
			gomega.Expect(name).To(gomega.Equal(""))
		})

		ginkgo.It("Should return error for invalid format (no slash)", func() {
			o := &Options{
				PlacementRef: "invalid-format",
			}

			_, _, err := o.parsePlacementRef()
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("must be in the format 'namespace/name'"))
		})

		ginkgo.It("Should return error for invalid format (too many slashes)", func() {
			o := &Options{
				PlacementRef: "namespace/name/extra",
			}

			_, _, err := o.parsePlacementRef()
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("must be in the format 'namespace/name'"))
		})

		ginkgo.It("Should return error for empty namespace", func() {
			o := &Options{
				PlacementRef: "/placement-name",
			}

			_, _, err := o.parsePlacementRef()
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("namespace and name cannot be empty"))
		})

		ginkgo.It("Should return error for empty name", func() {
			o := &Options{
				PlacementRef: "namespace/",
			}

			_, _, err := o.parsePlacementRef()
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("namespace and name cannot be empty"))
		})

		ginkgo.It("Should trim whitespace from namespace and name", func() {
			o := &Options{
				PlacementRef: " test-namespace / test-placement ",
			}

			namespace, name, err := o.parsePlacementRef()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(namespace).To(gomega.Equal("test-namespace"))
			gomega.Expect(name).To(gomega.Equal("test-placement"))
		})
	})

	ginkgo.Context("newClusterManagementAddon", func() {
		ginkgo.It("Should create ClusterManagementAddOn with Manual install strategy when placement-ref is not provided", func() {
			// Create a temporary manifest file
			tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer os.Remove(tmpFile.Name())

			manifestContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key: value`

			_, err = tmpFile.WriteString(manifestContent)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			tmpFile.Close()

			o := &Options{
				Name:          fmt.Sprintf("test-addon-%s", suffix),
				Version:       "0.0.1",
				PlacementRef:  "",
				Labels:        []string{},
				FileNameFlags: genericclioptions.FileNameFlags{Filenames: &[]string{tmpFile.Name()}, Recursive: ptr.To[bool](true)},
				Streams:       streams,
			}

			cma, err := newClusterManagementAddon(o)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(cma.Spec.InstallStrategy.Type).To(gomega.Equal(addonapiv1alpha1.AddonInstallStrategyManual))
			gomega.Expect(cma.Spec.InstallStrategy.Placements).To(gomega.BeNil())
		})

		ginkgo.It("Should create ClusterManagementAddOn with Placements install strategy when placement-ref is provided", func() {
			// Create a temporary manifest file
			tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer os.Remove(tmpFile.Name())

			manifestContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key: value`

			_, err = tmpFile.WriteString(manifestContent)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			tmpFile.Close()

			placementNamespace := fmt.Sprintf("placement-ns-%s", suffix)
			placementName := fmt.Sprintf("placement-%s", suffix)

			o := &Options{
				Name:          fmt.Sprintf("test-addon-%s", suffix),
				Version:       "0.0.1",
				PlacementRef:  fmt.Sprintf("%s/%s", placementNamespace, placementName),
				Labels:        []string{},
				FileNameFlags: genericclioptions.FileNameFlags{Filenames: &[]string{tmpFile.Name()}, Recursive: ptr.To[bool](true)},
				Streams:       streams,
			}

			cma, err := newClusterManagementAddon(o)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(cma.Spec.InstallStrategy.Type).To(gomega.Equal(addonapiv1alpha1.AddonInstallStrategyPlacements))
			gomega.Expect(cma.Spec.InstallStrategy.Placements).To(gomega.HaveLen(1))
			gomega.Expect(cma.Spec.InstallStrategy.Placements[0].Namespace).To(gomega.Equal(placementNamespace))
			gomega.Expect(cma.Spec.InstallStrategy.Placements[0].Name).To(gomega.Equal(placementName))
		})

		ginkgo.It("Should return error when placement-ref format is invalid", func() {
			// Create a temporary manifest file
			tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer os.Remove(tmpFile.Name())

			manifestContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key: value`

			_, err = tmpFile.WriteString(manifestContent)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			tmpFile.Close()

			o := &Options{
				Name:          fmt.Sprintf("test-addon-%s", suffix),
				Version:       "0.0.1",
				PlacementRef:  "invalid-format-no-slash",
				Labels:        []string{},
				FileNameFlags: genericclioptions.FileNameFlags{Filenames: &[]string{tmpFile.Name()}, Recursive: ptr.To[bool](true)},
				Streams:       streams,
			}

			_, err = newClusterManagementAddon(o)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("must be in the format 'namespace/name'"))
		})
	})

	ginkgo.Context("integration test with actual creation", func() {
		var addonName string

		ginkgo.AfterEach(func() {
			// Clean up ClusterManagementAddOn
			if addonName != "" {
				err = addonClient.AddonV1alpha1().ClusterManagementAddOns().Delete(
					context.Background(), addonName, metav1.DeleteOptions{})
				if err != nil && !errors.IsNotFound(err) {
					gomega.Expect(err).ToNot(gomega.HaveOccurred())
				}
			}

			// Clean up AddOnTemplate
			if addonName != "" {
				templateName := fmt.Sprintf("%s-0.0.1", addonName)
				err = addonClient.AddonV1alpha1().AddOnTemplates().Delete(
					context.Background(), templateName, metav1.DeleteOptions{})
				if err != nil && !errors.IsNotFound(err) {
					gomega.Expect(err).ToNot(gomega.HaveOccurred())
				}
			}
		})

		ginkgo.It("Should create ClusterManagementAddOn with placement reference", func() {
			addonName = fmt.Sprintf("test-addon-%s", suffix)
			placementNamespace := fmt.Sprintf("placement-ns-%s", suffix)
			placementName := fmt.Sprintf("placement-%s", suffix)

			// Create a temporary manifest file
			tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer os.Remove(tmpFile.Name())

			manifestContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key: value`

			_, err = tmpFile.WriteString(manifestContent)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			tmpFile.Close()

			o := &Options{
				ClusteradmFlags: testFlags,
				Name:            addonName,
				Version:         "0.0.1",
				PlacementRef:    fmt.Sprintf("%s/%s", placementNamespace, placementName),
				Labels:          []string{},
				FileNameFlags:   genericclioptions.FileNameFlags{Filenames: &[]string{tmpFile.Name()}, Recursive: ptr.To[bool](true)},
				Streams:         streams,
			}

			err = o.Run()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// Verify ClusterManagementAddOn was created with correct install strategy
			cma, err := addonClient.AddonV1alpha1().ClusterManagementAddOns().Get(
				context.Background(), addonName, metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(cma.Spec.InstallStrategy.Type).To(gomega.Equal(addonapiv1alpha1.AddonInstallStrategyPlacements))
			gomega.Expect(cma.Spec.InstallStrategy.Placements).To(gomega.HaveLen(1))
			gomega.Expect(cma.Spec.InstallStrategy.Placements[0].Namespace).To(gomega.Equal(placementNamespace))
			gomega.Expect(cma.Spec.InstallStrategy.Placements[0].Name).To(gomega.Equal(placementName))

			// Verify AddOnTemplate was created
			templateName := fmt.Sprintf("%s-0.0.1", addonName)
			template, err := addonClient.AddonV1alpha1().AddOnTemplates().Get(
				context.Background(), templateName, metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(template.Spec.AddonName).To(gomega.Equal(addonName))
		})

		ginkgo.It("Should create ClusterManagementAddOn without placement reference", func() {
			addonName = fmt.Sprintf("test-addon-no-placement-%s", suffix)

			// Create a temporary manifest file
			tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer os.Remove(tmpFile.Name())

			manifestContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key: value`

			_, err = tmpFile.WriteString(manifestContent)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			tmpFile.Close()

			o := &Options{
				ClusteradmFlags: testFlags,
				Name:            addonName,
				Version:         "0.0.1",
				PlacementRef:    "",
				Labels:          []string{},
				FileNameFlags:   genericclioptions.FileNameFlags{Filenames: &[]string{tmpFile.Name()}, Recursive: ptr.To[bool](true)},
				Streams:         streams,
			}

			err = o.Run()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// Verify ClusterManagementAddOn was created with Manual install strategy
			cma, err := addonClient.AddonV1alpha1().ClusterManagementAddOns().Get(
				context.Background(), addonName, metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(cma.Spec.InstallStrategy.Type).To(gomega.Equal(addonapiv1alpha1.AddonInstallStrategyManual))
			gomega.Expect(cma.Spec.InstallStrategy.Placements).To(gomega.BeNil())
		})
	})
})
