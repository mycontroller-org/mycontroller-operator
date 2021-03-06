module github.com/mycontroller-org/mycontroller-operator

go 1.16

require (
	github.com/mycontroller-org/server/v2 v2.0.0-20210801112728-7f593a7d9b08
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/openshift/api v0.0.0-20210713130143-be21c6cb1bea
	golang.org/x/tools v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/controller-runtime v0.9.2
)
