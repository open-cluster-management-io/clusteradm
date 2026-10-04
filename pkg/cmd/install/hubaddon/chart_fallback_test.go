// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

// The argocd-agent install falls back to the addon chart's own CRDs when the
// CRDs chart is missing from the repo. That relies on Helm's chart lookup
// returning an error that wraps repo.ErrNoChartName, so check it against the
// real lookup instead of a hand-built error.
func TestMissingCRDsChartMatchesErrNoChartName(t *testing.T) {
	dir := t.TempDir()
	repoConfig := filepath.Join(dir, "repositories.yaml")
	t.Setenv("HELM_REPOSITORY_CONFIG", repoConfig)
	t.Setenv("HELM_REPOSITORY_CACHE", dir)

	rf := repo.NewFile()
	rf.Update(&repo.Entry{Name: repoName, URL: "https://charts.invalid"})
	if err := rf.WriteFile(repoConfig, 0o600); err != nil {
		t.Fatal(err)
	}
	// An index without the CRDs chart, as in chart repos that predate it.
	if err := repo.NewIndexFile().WriteFile(filepath.Join(dir, repoName+"-index.yaml"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := (&action.ChartPathOptions{}).LocateChart(repoName+"/"+argocdAgentCRDsChart, cli.New())
	if err == nil {
		t.Fatal("expected an error for a chart missing from the repo index")
	}
	// helpers/helm InstallChart wraps the lookup error the same way.
	err = fmt.Errorf("failed to locate Helm chart: %w", err)
	if !errors.Is(err, repo.ErrNoChartName) {
		t.Fatalf("expected error to wrap repo.ErrNoChartName, got: %v", err)
	}
}
