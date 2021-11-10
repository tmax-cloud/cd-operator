package helmclient

import (
	"context"
	"os/exec"

	gohelm "github.com/mittwald/go-helm-client"
	cdexec "github.com/tmax-cloud/cd-operator/util/exec"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("helm-client")

type Client struct {
	Client gohelm.Client
}

// InstallChart installs helm chart
func (c *Client) InstallChart(chartSpec *gohelm.ChartSpec) (string, error) {
	// If the chart is already installed, trigger an upgrade instead.
	release, err := c.Client.InstallOrUpgradeChart(context.Background(), chartSpec)
	// if err != context.DeadlineExceeded { // TODO : 확인필요. ChartSpec wait true 문제 -> 왜 true로 set??
	if err != nil {
		return "", err
	}

	return release.Manifest, nil
}

// InstallChartByCLI installs helm chart by command line
func (c *Client) InstallChartByCLI(chartSpec *gohelm.ChartSpec) error {
	// TODO : 리소스들이 잘 배포되는데, Helm List에 안뜨는 이슈가 있음
	releaseName := chartSpec.ReleaseName
	chartLocalPath := chartSpec.ChartName
	namespace := chartSpec.Namespace

	// TODO : 범용적으로 쓰이게끔 리팩토링 할 것
	args := []string{"install", releaseName, chartLocalPath, "-n", namespace}
	stdout, err := cdexec.Run(exec.Command("helm", args...))

	if err != nil {
		return err
	}

	log.Info(stdout)
	return nil
}

// UninstallReleaseByName uninstalls a release identified by the provided 'name'.
func (c *Client) UninstallReleaseByName(releaseName string) error {
	if err := c.Client.UninstallReleaseByName(releaseName); err != nil {
		panic(err)
	}

	return nil
}
