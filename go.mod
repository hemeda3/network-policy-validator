module github.com/jbeda/tgik-controller

go 1.12

require (
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
)

replace (
	k8s.io/api => k8s.io/api v0.17.0

	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.3-beta.0
	k8s.io/client-go => k8s.io/client-go v0.17.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.0
	k8s.io/kubernetes => k8s.io/kubernetes v0.17.0
)

replace k8s.io/kubectl => k8s.io/kubectl v0.17.0

replace k8s.io/node-api => k8s.io/node-api v0.17.0
