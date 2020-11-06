# Kustomize SOPSGenerator

A Kustomize generator plugin that reads SOPS encoded files and converts them to Kubernetes Secrets

## Requirements

- Go 1.14 ([instructions](https://golang.org/doc/install))
- `kustomize` (`go get sigs.k8s.io/kustomize/kustomize/v3@v3.8.6`)
- Make
- [Mozilla's SOPS](https://github.com/mozilla/sops/)

## Building

Building is straightforward, just run `make` or `make build`

## Installation

It will install it into the first directory that it finds and that exists, in this order:

- `$KUSTOMIZE_PLUGIN_HOME`, if you have it set
- `$XDG_CONFIG_HOME/kustomize/plugin`, if `XDG_CONFIG_HOME` is set,
- `$HOME/.config/kustomize/plugin`
- `$HOME/kustomize/plugin`

For development, `$HOME/kustomize/plugin` is usually good enough, unless you explicitly want to use `$KUSTOMIZE_PLUGIN_HOME`

## Testing

### Adding the SOPS testing PGP secret and public keys to your keyring

To simplify testing, there's an ASCII armored PGP key pair provided in `__test__/pgp.asc`; import it into your PGP ring.

You need to know which PGP executable you're using, since some of them aren't compatible/aware of eachother. Assuming you are on
a relatively up-to-date system, you'll be using `gpg` by default, so this will suffice `gpg --import __test__/pgp.asc`. If you're
using `gpg2`, then use that, but remember which PGP executable you're using.

### Running tests

It should be as simple as running `make test`.

If you're using a different PGP executable, then you will need to [tell SOPS about it](https://github.com/mozilla/sops/#specify-a-different-gpg-executable),
you can do this by setting the `SOPS_GPG_EXEC` env variable either inline, or before running the tests:

```sh
export SOPS_GPG_EXEC=gpg2
make test
```

or

```sh
make SOPS_GPG_EXEC=gpg2 test
```

### Regenerating the test fixtures

Provided you followed the previous steps, you can add more fixtures to `__test__/plain` and run `make fixture` (same caveat about custom PGP executable applies).

## Usage

Currently, non-builtin plugins require you to use `kustomize` executable, _and_ to have it built from source. Installing Go is very easy, as is compiling
and installing Kustomize (you _need_ to run the `go get` command below _in this repo_).

```sh
go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4
```

That's it.

The reason for this is explained in more depth in [_A little about Go Kustomize plugins_](/docs/kustomize-plugins.md).

## Docker image

The docker image bundles kustomize and this plugin inside an Alpine Linux image.
The purpose of this is to allow/simplify kustomize with the plugin inside our GitLab-CI pipeline.

An example of how this might be used:

```yaml
job:
  image: registry.gitlab.com/mollybet/kustomize-sopsgenerator/kustomizer
  script:
    - kustomize build --enable_alpha_plugins path/to/kustomization -o output/dir
```

## API

The API is similar to that of the builtin `SecretGenerator`, the only thing that's not supported are `literals`, but that's on purpose, because then we'd have
to encode SOPS data with those, and we'd be deviating from the standard (as far as I know), which I wanted to avoid.

- `envs` - `[]string`: List of env files to expand as top-level entries in the secret.
  Env entries here take precedence over filename entries
- `files` - `[]string`: List of files to be included as secrets. The secret entry name will be the filename, and
  the value will be the contents. It's possible to rename the entry by doing `name=someFile.txt`. All files that are
  supported by SOPS natively are also supported here. In the "worst case" SOPS decays the type to "binary" so some niceties
  like unencrypted key-names are lost.

What's missing are `literals`, but there is no plan to support those for now, since they can easily be emulated using `.env`
files or `.ini` files.

`SecretGenerator` also supports "name-only" entries in `.env` files like `SOME_VARIABLE`, which is different from `SOME_VARIABLE=`.
The former signals `SecretGenerator` to read the value from the currently executing environment, and the latter means an empty value.

SOPS doesn't support this out of the box. We could add it via our own syntax, but I'd rather avoid deviating from the standard before
the need for it arises.

See the examples folder for a more concrete example. The examples assume you imported the testing PGP key.

### Logging

To get more useful logging information, you can add the `omninonsense.github.io/sopsgenerator.logLevel` annotation, it accepts one of the following
values (both sops and this plugin use [`logrus`](https://github.com/sirupsen/logrus)):

- `panic`
- `fatal`
- `error`
- (**default**) `warn`, or `warning`
- `info`
- `debug`
- `trace`

SOPS logs some failures as... Info. So, when in doubt, set log level to `debug`, it will at least point you in the right direction.

## TODO

- More tests
- Don't use kustomize's internal testing framework, since it wants things inside `~/`
- Consider switching to an executor plugin? Go's compilation skew might become an annoying problem in the future


[golang/go!17150]: https://github.com/golang/go/issues/17150
[golang/go!24034]: https://github.com/golang/go/issues/24034
