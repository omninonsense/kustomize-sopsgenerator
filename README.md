# Kustomize SOPSGenerator

A Kustomize generator plugin that reads SOPS encoded files and converts them to Kubernetes Secrets

<details>
<summary><strong>Why SOPS at all?</strong></summary>

Another option that was considered was Hashicorp's Vault, but Vault is a lot of setup:

- Vault cluster
- Consul cluster
- a load balancer
- AWS VPCs
- unsealing via Shamit secret sharing
  - manual/operator based
  - could be automated with AWS KMS, though
- S3 for storing secrets (or something else)
- AWS AMIs
- AWS IAMs
- Consul ACLS
- Vault ACLs
- … and the rest of the alphabet

None of these are unsurmountable, they're all documented, and Vault is a great piece of software,
but the main problem is that our current workflow and setup is radically different from using something like Vault.
So switching to Vault would be a huge technological and mental overhead for everyone.

So, while looking for a middle ground, I found SOPS. We still get to keep our secrets in git, our deployments won't change too much
things will more or less have the same access to secrets as they had before, and we get to remove Ansible and Ansible Vault from
our workflow/deployment while simplifying it.

Additionally, SOPS can publish secrets to Vault, so if we want to migrate to Vault or use it alongside SOPS, we have the option to
do it gradually, or on a per-needed basis.

No, `core/secrets` wasn't considered.

</details>

## Requirements

- Go 1.13 ([instructions](https://golang.org/doc/install)) (version is important)
- `kustomize` (`go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4`)
- Make
- [Mozilla's SOPS](https://github.com/mozilla/sops/)

## Building

Building is straightforward, just run `make` or `make build`

## Testing

Testing is also straightforward, but there are a few extra steps that you need to do initially, this is to simplify testing.

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

Currently, non-builtin plugins require you to use `kustomize` executable, _and_ to have kustomize built from source. Installing Go is very easy, as is compiling
compiling and installing Kustomize:

```sh
go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4
```

That's it.

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

## TODO

- Discuss with team to use [`go:generate`](https://golang.org/pkg/cmd/go/internal/generate/) instead of Makefiles, if they're comfortable with it, or at least for some tasks since it integrates better with Go.
- More tests?
