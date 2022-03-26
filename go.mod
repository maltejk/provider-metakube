module github.com/maltejk/provider-metakube

go 1.16

require (
	github.com/crossplane/crossplane-runtime v0.15.0
	github.com/crossplane/crossplane-tools v0.0.0-20210320162312-1baca298c527
	github.com/crossplane/provider-template v0.0.0-20211217231306-2f40be13c7b8
	github.com/google/go-cmp v0.5.6
	github.com/maltejk/metakube-go-client v0.0.0-20220326100423-a21f7fa21581 // indirect
	github.com/pkg/errors v0.9.1
	golang.org/x/net v0.0.0-20220325170049-de3da57026de // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	sigs.k8s.io/controller-runtime v0.9.6
	sigs.k8s.io/controller-tools v0.6.2
)
