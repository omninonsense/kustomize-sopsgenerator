BUILDDIR := build

# The key fingerprint from __test__/test_pgp.asc
export SOPS_PGP_FP := F4F835FE4A069B4025A1000F896B05FFB977131D

PLUGIN_DIR=$(shell go run SOPSGenerator.go build_helpers.go subdir)
build: clean
	mkdir -p $(BUILDDIR)/$(PLUGIN_DIR)
	go build -buildmode=plugin -o $(BUILDDIR)/$(PLUGIN_DIR)/SOPSGenerator.so

clean:
	rm -rfv build/

test:
	go test

# Encrypt files in __test__/plain.
# TODO: Simplify this, if possible, or maybe just make it a shell script?
FILENAME := $(patsubst __test__/plain/%,__test__/encrypted/%,$(wildcard __test__/plain/*))
fixtures: clean-fixtures $(FILENAME)
__test__/encrypted/%: __test__/plain/%
	sops --output $@ --encrypt $<
clean-fixtures:
	rm -f __test__/encrypted/*
