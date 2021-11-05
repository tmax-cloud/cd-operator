package helmclient

import (
	"context"

	gohelm "github.com/mittwald/go-helm-client"
)

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

// UninstallReleaseByName uninstalls a release identified by the provided 'name'.
func (c *Client) UninstallReleaseByName(releaseName string) error {
	if err := c.Client.UninstallReleaseByName(releaseName); err != nil {
		panic(err)
	}

	return nil
}
