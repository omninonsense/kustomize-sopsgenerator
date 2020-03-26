// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	sg "omninonsense.github.io/kustomize-sopsgenerator"

	kusttest_test "sigs.k8s.io/kustomize/api/testutils/kusttest"
)

// TODO: Write more tests
func TestSOPSGeneratorPlugin(t *testing.T) {
	th := kusttest_test.
		MakeEnhancedHarness(t).
		BuildGoPlugin(sg.Domain, sg.Version, sg.Kind)

	defer th.Reset()

	json, _ := ioutil.ReadFile("__test__/encrypted/secret.json")
	yaml, _ := ioutil.ReadFile("__test__/encrypted/secret.yaml")
	env, _ := ioutil.ReadFile("__test__/encrypted/secret.env")

	th.WriteF("secret.json", string(json))
	th.WriteF("secret.yaml", string(yaml))
	th.WriteF("secret.env", string(env))

	m := th.LoadAndRunGenerator(fmt.Sprintf(`
apiVersion: %s/%s
kind: %s
metadata:
  name: zero-zero-seven
  namespace: test
envs:
  - secret.env
files:
  - secret.json
  - renamed.yaml=secret.yaml
`, sg.Domain, sg.Version, sg.Kind))

	th.AssertActualEqualsExpected(m, `
apiVersion: v1
data:
  EMPTY: ""
  HELLO: d29ybGQ=
  renamed.yaml: aGVsbG86IHdvcmxkCg==
  secret.json: ewoJImhlbGxvIjogIndvcmxkIgp9
kind: Secret
metadata:
  name: zero-zero-seven
  namespace: test
type: Opaque
`)
}
