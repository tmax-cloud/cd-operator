package helmclient

import (
	"context"

	gohelm "github.com/mittwald/go-helm-client"
)

type Client struct {
	Client gohelm.Client
}

// InstallChart installs helm chart
func (c *Client) InstallChart(releaseName, chartDir, namespace string) error {
	// Use an unpacked(locally cloned) chart directory
	chartSpec := gohelm.ChartSpec{
		ReleaseName: releaseName,
		ChartName:   chartDir,
		Namespace:   namespace,
		UpgradeCRDs: true,
		Wait:        true,
	}

	// If the chart is already installed, trigger an upgrade instead.
	_, err := c.Client.InstallOrUpgradeChart(context.Background(), &chartSpec)

	if err != context.DeadlineExceeded { // TODO : 확인필요. 리소스들도 정상적으로 다 생기는데, 왜 이게 발생하는 걸까?
		if err != nil {
			panic(err)
		}
	}

	return nil
}

// UninstallReleaseByName uninstalls a release identified by the provided 'name'.
func (c *Client) UninstallReleaseByName(releaseName string) error {
	if err := c.Client.UninstallReleaseByName(releaseName); err != nil {
		panic(err)
	}

	return nil
}
