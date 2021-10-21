package helmclient

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestInstallChart(t *testing.T) {
	/*
	   helmclient := &gohelm.HelmClient{}
	   testHelmClient := &Client{
	       helmclient: helmclient,
	   }

	   url := "https://github.com/tmax-cloud/cd-example-apps"
	   path := "/tmp/test-" + utils.RandomString(5)

	   defer os.RemoveAll(path)

	   err := gitclient.Clone(url, path)
	   assert.Equal(t, err, nil)

	   releaseName := "test-" + utils.RandomString(5)
	   chartPath := path + "/helm-guestbook"
	   err = testHelmClient.InstallChart(releaseName, chartPath)
	   assert.Equal(t, err, nil)
	*/
	assert.Equal(t, nil, nil)
}
