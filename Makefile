BUILD_DIR := build
KIND := $(shell go run SOPSGenerator.go build_helpers.go kind)
PLUGIN_DIR := $(shell go run SOPSGenerator.go build_helpers.go subdir)

# The key fingerprint from __test__/test_pgp.asc
export SOPS_PGP_FP := F4F835FE4A069B4025A1000F896B05FFB977131D

build: clean
	mkdir -p $(BUILD_DIR)/$(PLUGIN_DIR)
	go build -buildmode=plugin -o $(BUILD_DIR)/$(PLUGIN_DIR)/$(KIND).so

clean:
	rm -rfv $(BUILD_DIR)/

test:
	@echo "The Kustomize testing framwork still uses old locations for running tests instead of new ones"
	@echo "So the tests will create ~/sigs.k8s.io/kustomize/ and build the the plugin into it before"
	@echo "running the tests."
	@echo
	mkdir -p ${HOME}/sigs.k8s.io/kustomize/plugin
	make clean build BUILD_DIR=${HOME}/sigs.k8s.io/kustomize/plugin
	go test

install: $(BUILD_DIR)/$(PLUGIN_DIR)/$(KIND).so
	install $< -D $(shell go run SOPSGenerator.go build_helpers.go plugin-home)/$(PLUGIN_DIR)/$(KIND).so

# Encrypt files in __test__/plain.
# TODO: Simplify this, if possible, or maybe just make it a shell script?
FILENAME := $(patsubst __test__/plain/%,__test__/encrypted/%,$(wildcard __test__/plain/*))
fixtures: clean-fixtures $(FILENAME)
__test__/encrypted/%: __test__/plain/%
	sops --output $@ --encrypt $<
clean-fixtures:
	rm -f __test__/encrypted/*
