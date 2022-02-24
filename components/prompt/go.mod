module github.com/RafaySystems/ztka/components/prompt

go 1.13

require (
	github.com/RafaySystems/rafay-common v1.6.0-beta3.0.20210803105956-3fbeb5be8ae9
	github.com/RafaySystems/rcloud-base/components/common v0.0.0-00010101000000-000000000000
	github.com/RafaySystems/rctl v1.6.2
	github.com/Shopify/sarama v1.29.1 // indirect
	github.com/creack/pty v1.1.11
	github.com/gorilla/websocket v1.4.2
	github.com/julienschmidt/httprouter v1.3.0
	github.com/mattn/go-runewidth v0.0.8
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f
	github.com/pkg/term v0.0.0-20180423043932-cda20d4ac917
	github.com/rs/xid v1.3.0
	github.com/spf13/viper v1.7.1
	github.com/urfave/negroni v1.0.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/sys v0.0.0-20211123173158-ef496fb156ab
	k8s.io/api v1.16.4
	k8s.io/apimachinery v1.16.4
	k8s.io/client-go v0.23.4
	sigs.k8s.io/controller-runtime v0.11.1
)

replace (
	github.com/go-pg/pg => github.com/go-pg/pg v6.15.1+incompatible
	k8s.io/api => k8s.io/api v0.21.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.1-rc.0
	k8s.io/apiserver => k8s.io/apiserver v0.21.10
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.10
	k8s.io/client-go => k8s.io/client-go v0.21.10
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.10
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.10
	k8s.io/code-generator => k8s.io/code-generator v0.21.10
	k8s.io/component-base => k8s.io/component-base v0.21.10
	k8s.io/cri-api => k8s.io/cri-api v0.21.10
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.10
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.10
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.10
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.10
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.10
	k8s.io/kubectl => k8s.io/kubectl v0.21.10
	k8s.io/kubelet => k8s.io/kubelet v0.21.10
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.10
	k8s.io/metrics => k8s.io/metrics v0.21.10
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.10
)
