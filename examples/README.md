# Examples

The examples assume you've read the README.

Before we start, make sure you've ran `make build` and that your CWD is this (examples) directory.

Next, let's set `KUSTOMIZE_PLUGIN_HOME` to the `build` directory:

```sh
export KUSTOMIZE_PLUGIN_HOME=$(readlink -f ../build)
```

## Database example

Run `kustomize build --enable_alpha_plugins database/`
