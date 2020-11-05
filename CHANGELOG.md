# Changelog

## v1.1.3

Fixes:

- Pass `options` to the underlying `SecretGenerator` (e138f1d)

## v1.1.2

Fixes:

- Error when we see `literals` in the generator; we're not processing them, and it's better than silently ignoring them (6e84a44)
- Refactor: Use in-memory FS when cahining to `SecretGenerator` (see #3) (79c1ac2)

## v1.1.1

Fixes:

- Dockerfile: don't use `kustomize` as an entry point in the Docker image (0b33e49)

## v1.1.0

Features:

- `omninonsense.github.io/sopsgenerator.logLevel` annotation to control logging and aid in debugging (2c7b967)

## v1.0.0

Initial release
