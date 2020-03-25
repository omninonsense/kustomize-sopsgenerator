# Kustomize SOPSGenerator

A Kustomize generator plugin that reads SOPS encoded files and converts them to Kubernetes Secrets

## Requirements

- at least Go 1.14 ([instructions](https://golang.org/doc/install))
- Make
- [Mozilla's SOPS](https://github.com/mozilla/sops/)

## Building

Building is straightforward, just run `make` or `make build`

## Testing

Testing is also straightforward, but there are a few extra steps that you need to do initially, this is to simplify testing.

### Adding the SOPS testing PGP secret and public keys to your keyring

To simplify testing, there's an ASCII armored PGP key pair provided in `__test__/pgp.asc`; import it into your PGP ring.

You need to know which PGP executable you're using, since some of them aren't compatible/aware of eachother. Assuming you are on
a relatively up-to-date system, you'll be using `gpg` by default, so this will suffice `gpg --import __test__/pgp.test`. If you're
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

## A little about Go Kustomize plugins

Kustomize looks for plugins, in the following order ([ref](https://sigs.k8s.io/kustomize/api/konfig/plugins.go#L69-L102)):

- `$KUSTOMIZE_PLUGIN_HOME/{domain}/{version}/{lowercase kind}/{kind}.so`
- `$XDG_CONFIG_HOME/kustomize/{domain}/{version}/{lowercase kind}/{kind}.so`
- `$HOME/.config/kustomize/{domain}/{version}/{lowercase kind}/{kind}.so`
- `$HOME/kustomize/{domain}/{version}/{lowercase kind}/{kind}.so`

**Example**: Let's say we've set `KUSTOMIZE_PLUGIN_HOME=/home/cirunner/molly/k8splugin`, and that we have the following plugin configuration:

- Kind: `SOPSGenerator`
- Domain: `omninonsense.github.com` (**TODO** change to something like `k8s.mollybet.com` before releasing)
- Version: `v1beta`

our plugin could be in `/home/cirunner/molly/k8splugin/omninonsense.github.com/v1beta/sopsgenerator/SOPSGenerator.so`.

It has a similar algorithm for looking up "exec plugins" (scripts, but they're not as flexible), in fact, it first tried to look up an exec plugin, and checks for
a [Go Plugin](https://golang.org/pkg/plugin/).

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

I thing I went trhrough all of them, but none of them satisfied me in terms of quality.

There was a promising [one](https://github.com/goabout/kustomize-sopssecretgenerator) written in Go, but it was an _exec_ "plugin"
that was concidentally written in Go, it wasn't an actual _Go plugin_. So, it didn't benefit from anything mentioned in the
previous question, and was plagued by all the problems of exec plugins.

A few others float out there, but they are examples or proof-of-concepts that din't look production ready, and had either awkward
APIs that didn't resemble the other Kustomize APIs, or were incomplete.

</details>

<details>
<summary><strong>Why SOPS at all?</strong></summary>

Another option that was consideted was Hashicorp's Vault, but Vault is a lot of setup:

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

## Usage

### Example

Best to kick it off with an example:

<details>
<summary><code>postgres.env</code> contents</summary>

```env
DBNAME=molly
PGPASS=hug3M0n3y
PGUSER=webapp
```

</details>

<details>
<summary><code>smtp.env</code> contents</summary>

```env
SES_USERNAME=ses-molly-mailer
# Yes, we're reusing passwords; deal with it
SES_PASSWORD=hug3M0n3y
SES_SERVE=sas.aws.com
```

</details>

The kustomization resource:

```yaml
apiVersion: omninonsense.github.io/v1beta
kind: SOPSGenerator
metadata:
  name: top-secret
  namespace: production
envs:
  - postgres.env
  - smtp.env
files:
  - credentials.yaml
  - shorter.json=some-really-long-name-that-nobody-remembers.json
```

You might've noticed that the API is similar to that of the builtin `SecretGenerator`, the only thing that's not
supported are `literals`, but that's on purpose, because then we'd have to encode SOPS data with those, and we'd
be deviating from the standard (as far as I know), which I wanted to avoid.

The above example will produce this—pseudo-code version version of a—Kubernetes secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: zero-zero-seven
  namespace: test
type: Opaque
data:
  DBNAME: molly
  PGPASS: hug3M0n3y
  PGUSER: webapp
  SES_USERNAME: ses-molly-mailer
  SES_PASSWORD: hug3M0n3y
  SES_SERVE: sas.aws.com
  credentials.yaml: (... contents of credentials.yaml)
  shorter.json: (... contents of some-really-long-name-that-nobody-remembers.json)
```

The values would be base64 encoded for real though. Also in real wouldn't be exposing the file contents at all,
we would just be forwarding it to kubernetes directly, but sometimes it's useful to see the contents for debugging.

### API

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

## TODO

- Discuss with team to use [`go:generate`](https://golang.org/pkg/cmd/go/internal/generate/) instead of Makefiles, if they're comfortable with it, or at least for some tasks since it integrates better with Go.

- More tests?
