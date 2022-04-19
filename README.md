# this is all non-working code

`provider-metakube` is a minimal [Crossplane](https://crossplane.io/) Provider
 for SysEleven's Metakube. It is work in progress and not yet feature complete.
 Please see the CRDs for resources currently supported.

## Developing

1. Use this repository as a template to create a new one.
1. Find-and-replace `provider-template` with your provider's name.
1. Run `make` to initialize the "build" Make submodule we use for CI/CD.
1. Run `make reviewable` to run code generation, linters, and tests.
1. Replace `Project` with your own managed resource implementation(s).

Refer to Crossplane's [CONTRIBUTING.md] file for more information on how the
Crossplane community prefers to work. The [Provider Development][provider-dev]
guide may also be of use.

[CONTRIBUTING.md]: https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md
[provider-dev]: https://github.com/crossplane/crossplane/blob/master/docs/contributing/provider_development_guide.md