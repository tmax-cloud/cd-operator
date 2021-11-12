package utils

import (
	"log"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type bytesToUnstructuredObjectTestCase struct {
	bytes []byte

	expectedObj      map[string]interface{}
	expectedErrOccur bool
	expectedErrMsg   string
}

func TestBytesToUnstructuredObject(t *testing.T) {
	if os.Getenv("CI") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	tc := map[string]bytesToUnstructuredObjectTestCase{
		"success": {
			bytes:            []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"test-deploy-from-git"},"spec":{"selector":{"matchLabels":{"apps":"test-app"}},"template":{"metadata":{"labels":{"apps":"test-app"},"name":"nginx"},"spec":{"containers":[{"image":"nginx","name":"nginx-container","ports":[{"containerPort":80}]}]}}}}`),
			expectedErrOccur: false,
			expectedObj:      map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment", "metadata": map[string]interface{}{"name": "test-deploy-from-git"}, "spec": map[string]interface{}{"selector": map[string]interface{}{"matchLabels": map[string]interface{}{"apps": "test-app"}}, "template": map[string]interface{}{"metadata": map[string]interface{}{"labels": map[string]interface{}{"apps": "test-app"}, "name": "nginx"}, "spec": map[string]interface{}{"containers": []interface{}{map[string]interface{}{"image": "nginx", "name": "nginx-container", "ports": []interface{}{map[string]interface{}{"containerPort": int64(80)}}}}}}}},
		},
		"noJSONByteFail": {
			bytes: []byte(`apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: test-deploy-from-git
			spec:
			  template:
				metadata:
				  name: nginx
				  labels:
					apps: test-app
				spec:
				  containers:
					- name: nginx-container
					  image: nginx
					  ports:
						- containerPort: 80
			  selector:
				matchLabels:
				  apps: test-app`),

			expectedErrOccur: true,
			expectedErrMsg:   "error decoding number from json: invalid character 'a' looking for beginning of value",
		},
	}
	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			unstObj, err := BytesToUnstructuredObject(c.bytes)
			if c.expectedErrOccur {
				require.Error(t, err)
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, c.expectedObj, unstObj.Object)
			}
		})
	}
}

type splitMultipleObjectsYAMLTestCase struct {
	rawYAML []byte

	expectedObjCnt int
}

func TestSplitMultipleObjectsYAML(t *testing.T) {
	if os.Getenv("CI") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	tc := map[string]splitMultipleObjectsYAMLTestCase{
		"oneObject": {
			rawYAML: []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy-from-git
spec:
  template:
	metadata:
	  name: nginx
	  labels:
		apps: test-app
	spec:
	  containers:
		- name: nginx-container
		  image: nginx
		  ports:
			- containerPort: 80
  selector:
	matchLabels:
	  apps: test-app
`),
			expectedObjCnt: 1,
		},
		"objectWith---": {
			rawYAML: []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy-from-git
spec:
  template:
	metadata:
	  name: nginx
	  labels:
		apps: ---
	spec:
	  containers:
		- name: nginx-container
		  image: nginx
		  ports:
			- containerPort: 80
  selector:
	matchLabels:
	  apps: test-app
`),
			expectedObjCnt: 1,
		},
		"noJSONByteFail": {
			rawYAML: []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy-from-git
spec:
  template:
	metadata:
	  name: nginx
	  labels:
		apps: test-app
	spec:
	  containers:
		- name: nginx-container
		  image: nginx
		  ports:
			- containerPort: 80
  selector:
	matchLabels:
	  apps: test-app
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy-from-git2
spec:
  template:
	metadata:
	  name: nginx
	  labels:
		apps: test-app
	spec:
	  containers:
		- name: nginx-container
		  image: nginx
		  ports:
			- containerPort: 80
  selector:
	matchLabels:
	  apps: test-app
			`),
			expectedObjCnt: 2,
		},
	}
	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			objects := SplitMultipleObjectsYAML(c.rawYAML)

			log.Println(objects)
			require.Equal(t, c.expectedObjCnt, len(objects))
		})
	}
}
