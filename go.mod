module github.com/tmax-cloud/cd-operator

go 1.15

require (
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/go-git/go-git/v5 v5.4.2
	github.com/go-logr/logr v0.3.0
	github.com/gorilla/mux v1.8.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/operator-framework/operator-lib v0.1.0
	github.com/sirupsen/logrus v1.8.1
	github.com/sourcegraph/go-diff v0.6.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/tools v0.0.0-20200916195026-c9a70fc28ce3 // indirect
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/kube-aggregator v0.18.8
	k8s.io/kubernetes v1.13.0
	knative.dev/pkg v0.0.0-20200922164940-4bf40ad82aab
	sigs.k8s.io/controller-runtime v0.6.4
	sigs.k8s.io/yaml v1.2.0
)

replace (
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200922164940-4bf40ad82aab
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.4
)

// Pin k8s deps to v0.18.8
replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
)
