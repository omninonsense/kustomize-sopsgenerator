# A little about Go Kustomize plugins

## Locations

Kustomize looks for plugins, in the following order ([ref](https://sigs.k8s.io/kustomize/api/konfig/plugins.go#L69-L102)):

- `$KUSTOMIZE_PLUGIN_HOME/{domain}/{version}/{lowercase kind}/{kind}.so`
- `$XDG_CONFIG_HOME/kustomize/{domain}/{version}/{lowercase kind}/{kind}.so`
- `$HOME/.config/kustomize/{domain}/{version}/{lowercase kind}/{kind}.so`
- `$HOME/kustomize/{domain}/{version}/{lowercase kind}/{kind}.so`

**Example**: Let's say we've set `KUSTOMIZE_PLUGIN_HOME=/k8splugin`, and that we have the following plugin configuration:

- Kind: `SOPSGenerator`
- Domain (Group): `omninonsense.github.com`
- Version: `v1beta`

our plugin would be in `/k8splugin/omninonsense.github.com/v1beta/sopsgenerator/SOPSGenerator.so`.

<details>
<summary><strong>Why do I have to insall Go and build Kustomize to use this?</strong></summary>

It boils down to two reason:

The first one is that `kubectl` is slow with adding/updating features, so the version of the Kustomize API in
it is a bit older

The second reason is that, even if you were to download the latest `kustomize` from the GitHub's release page, it
is compiled by Go with `CGO_ENABLED=0`, which means that the go `plugin` package is compiled with with a stub implementation,
which just returns the error: `plugin: not implemented` ([source](https://golang.org/src/plugin/plugin_stubs.go)).

This is probably a sidefect of wanting to avoid compilation skew problems associated dynamic linking?

</details>
