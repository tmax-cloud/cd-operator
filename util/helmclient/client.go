package helmclient

/*
type Client struct {
    helmclient *gohelm.HelmClient
}
*/

/*
type Client struct {
    helmclient *gohelm.HelmClient
}
*/

// InstallChart installs helm chart
func InstallChart(releaseName, chartDir string) error {
	/*
		// Use an unpacked(locally cloned) chart directory
		chartSpec := gohelm.ChartSpec{
			ReleaseName: releaseName,
			ChartName:   chartDir,
			Namespace:   "default",
			UpgradeCRDs: true,
			Wait:        true,
		}

		_, err := c.helmclient.InstallOrUpgradeChart(context.Background(), &chartSpec)

		if err != nil {
			panic(err)
		}

		return err
	*/
	return nil
}
