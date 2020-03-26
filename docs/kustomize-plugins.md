# A little about Go Kustomize plugins

## Locations

Kustomize looks for plugins, in the following order ([ref](https://sigs.k8s.io/kustomize/api/konfig/plugins.go#L69-L102)):

- `$KUSTOMIZE_PLUGIN_HOME/{domain}/{version}/{lowercase kind}/{kind}.so`
- `$XDG_CONFIG_HOME/kustomize/{domain}/{version}/{lowercase kind}/{kind}.so`
- `$HOME/.config/kustomize/{domain}/{version}/{lowercase kind}/{kind}.so`
- `$HOME/kustomize/{domain}/{version}/{lowercase kind}/{kind}.so`

**Example**: Let's say we've set `KUSTOMIZE_PLUGIN_HOME=/home/cirunner/molly/k8splugin`, and that we have the following plugin configuration:

- Kind: `SOPSGenerator`
- Domain (Group): `omninonsense.github.com`
- Version: `v1beta`

our plugin could be in `/home/cirunner/molly/k8splugin/omninonsense.github.com/v1beta/sopsgenerator/SOPSGenerator.so`.

It has a similar algorithm for looking up "exec plugins" (scripts, but they're not as flexible), in fact, it first tried to look up an exec plugin, and checks for
a [Go Plugin](https://golang.org/pkg/plugin/).

## FAQs

<details>
<summary><strong>Why was this written in Go?</strong></summary>

Mainly because Kubernetes and Kustomize are written in Go, and this uses the same APIs as them.
So, this gives us the benefit of safety by letting Kubernetes and Kustomize load, compose, and generate its own resources.
No fiddly file generation and templating.

Also, it's easier to debug and test these plugins than their exec equivalents.

It does have problems like compilation skew due to different architectures, but those can be solved by compiling for multiple targets,
if we have this problem in the future.

</details>

<details>
<summary><strong>Why not use an existing Kustomize+SOPS plugin?</strong></summary>

I thing I went through most (if not all) of them, but none of them satisfied me in terms of quality.

There was a promising [one](https://github.com/goabout/kustomize-sopssecretgenerator) written in Go, but it was an _exec_ "plugin"
that was coincidentally written in Go, it wasn't an actual _Go plugin_. So, it didn't benefit from anything mentioned in the
previous question, and was plagued by all the problems of exec plugins.

A few others float out there, but they are examples or proof-of-concepts that didn't look production ready, and had either awkward
APIs that didn't resemble the other Kustomize APIs, or were incomplete.

</details>

<details>
<summary><strong>Why do I have to insall Go and build Kustomize to use this?</strong></summary>

It boils down to two reason:

The first one is that `kubectl` is slow with adding/updating features, so the version of the Kustomize API in
it is a bit older

The second reason is that, even if you were to download the latest `kustomize` from the GitHub's release page, it
is compiled by Go with `CGO_ENABLED=0`, which means that the go `plugin` package is compiled with with a stub implementation,
which just returns the error: `plugin: not implemented` ([source](https://golang.org/src/plugin/plugin_stubs.go)).

This is probably a sidefect of wanting to avoid provenance and compilation skew problems associated dynamic linking?

</details>
