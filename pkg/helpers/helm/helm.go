// Copyright Contributors to the Open Cluster Management project
package helm

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/gofrs/flock"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

const HelmFlagSetAnnotation = "HelmSet"

type Helm struct {
	settings *cli.EnvSettings
	values   *values.Options
}

func NewHelm() *Helm {
	h := &Helm{
		settings: cli.New(),
		values: &values.Options{
			Values:     []string{},
			FileValues: []string{},
		},
	}
	return h
}

func (h *Helm) WithNamespace(ns string) {
	h.settings.SetNamespace(ns)
}

func (h *Helm) AddFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&h.values.ValueFiles, "values", "f", []string{}, "specify values in a YAML file")
	fs.StringArrayVar(&h.values.Values, "set-string", []string{}, "set string for chart")
	fs.StringArrayVar(&h.values.Values, "set", []string{}, "set values for chart")
	fs.StringArrayVar(&h.values.FileValues, "set-file", []string{}, "set file for chart")
	fs.StringArrayVar(&h.values.JSONValues, "set-json", []string{}, "set json for chart")
	fs.StringArrayVar(&h.values.LiteralValues, "set-literal", []string{}, "set literal for chart")
	_ = fs.SetAnnotation("values", "HelmSet", []string{})
	_ = fs.SetAnnotation("set-string", "HelmSet", []string{})
	_ = fs.SetAnnotation("set", "HelmSet", []string{})
	_ = fs.SetAnnotation("set-file", "HelmSet", []string{})
	_ = fs.SetAnnotation("set-json", "HelmSet", []string{})
	_ = fs.SetAnnotation("set-literal", "HelmSet", []string{})
}

func (h *Helm) SetValue(key, value string) {
	h.values.Values = append(h.values.Values, fmt.Sprintf("%s=%s", key, value))
}

// PrepareChart prepares the chart for installation
func (h *Helm) PrepareChart(repoName, repoUrl string) error {
	// add repo
	repoFile := h.settings.RepositoryConfig

	//Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer func() {
			err := fileLock.Unlock()
			if err != nil {
				log.Fatal(err)
			}
		}()
	}
	if err != nil {
		log.Fatal(err)
	}

	b, err := os.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		log.Fatal(err)
	}

	//if repo not exist, add it
	if !f.Has(repoName) {
		c := repo.Entry{
			Name: repoName,
			URL:  repoUrl,
		}

		r, err := repo.NewChartRepository(&c, getter.All(h.settings))
		if err != nil {
			log.Fatal(err)
		}

		if _, err := r.DownloadIndexFile(); err != nil {
			err := errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", repoUrl)
			log.Fatal(err)
		}

		f.Update(&c)

		if err := f.WriteFile(repoFile, 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%q has been added to your repositories\n", repoName)
	}

	// update repo
	var ocmRepo *repo.ChartRepository
	for _, cfg := range f.Repositories {
		if cfg.Name == repoName {
			r, err := repo.NewChartRepository(cfg, getter.All(h.settings))
			if err != nil {
				return err
			}
			ocmRepo = r
		}
	}
	fmt.Printf("Hang tight while we grab the latest from ocm chart repository...\n")

	if _, err := ocmRepo.DownloadIndexFile(); err != nil {
		return fmt.Errorf("unable to get an update from the %q chart repository (%s):\n\t%s", ocmRepo.Config.Name, ocmRepo.Config.URL, err)
	}
	fmt.Printf("Successfully got an update from the %q chart repository\n", ocmRepo.Config.Name)
	return nil
}

// InstallChart installs the chart
func (h *Helm) InstallChart(name, repo, chart string) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(h.settings.RESTClientGetter(), h.settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
		log.Fatal(err)
	}
	client := action.NewInstall(actionConfig)

	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}
	client.ReleaseName = name
	cp, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", repo, chart), h.settings)
	if err != nil {
		log.Fatal(err)
	}

	debug("CHART PATH: %s\n", cp)

	p := getter.All(h.settings)
	vals, err := h.values.MergeValues(p)
	if err != nil {
		log.Fatal(err)
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		log.Fatal(err)
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		log.Fatal(err)
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: h.settings.RepositoryConfig,
					RepositoryCache:  h.settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatal(err)
			}
		}
	}

	client.Namespace = h.settings.Namespace()
	release, err := client.Run(chartRequested, vals)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(release.Manifest)
}

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func debug(format string, v ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
	_ = log.Output(2, fmt.Sprintf(format, v...))
}
