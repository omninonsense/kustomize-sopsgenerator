package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"path/filepath"

	"github.com/pkg/errors"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/decrypt"

	logrus "github.com/sirupsen/logrus"
	sopsLogging "go.mozilla.org/sops/v3/logging"

	"sigs.k8s.io/kustomize/api/ifc"
	"sigs.k8s.io/kustomize/api/kv"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

type PluginMeta struct {
	types.ObjectMeta `json:",inline,omitempty" yaml:",inline,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// A secret generator that reads SOPS encoded secrets and feeds them to a secreteGenerator
type plugin struct {
	h          *resmap.PluginHelpers
	PluginMeta `json:"metadata,omitempty" yaml:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	types.SecretArgs
	types.GeneratorOptions
}

//nolint: deadcode
// The domain of the plugin; this is something Kubernetes wants, I've just used my github username for now
const Domain = "omninonsense.github.io"

//nolint: deadcode
// The version of the plugin
const Version = "v1beta"

//nolint: deadcode
// The Kind of the Kubernetes Resource (or in the case of a plugin, a Kustomization resource).
const Kind = "SOPSGenerator"

type kvMap map[string]string

//nolint: unused,deadcode
var KustomizePlugin plugin

var utf8bom = []byte{0xEF, 0xBB, 0xBF}

// We use the SOPS wrapped logrus, so we can share loglevels with them
var log = sopsLogging.NewLogger("SOPSGenerator")

func sopsGenErr(err error) error {
	log.Error(err)
	return err
}

func fmtAnnotationName(annot string) string {
	return fmt.Sprintf("%s/%s.%s", Domain, strings.ToLower(Kind), annot)
}

// Called by Kustomize to prime/configure the plugin with the YAML file
// It determines which plugin to load using `Kind` and `Version`.
func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h

	if err := yaml.Unmarshal(c, p); err != nil {
		return err
	}

	logLevelAnnotation := fmtAnnotationName("logLevel")

	if name, ok := p.PluginMeta.Annotations[logLevelAnnotation]; ok {
		level, err := logrus.ParseLevel(name)

		if err != nil {
			log.Error(err)
			return err
		} else {
			sopsLogging.SetLevel(level)
		}

	}

	return nil
}

//
func (p *plugin) Generate() (resmap.ResMap, error) {
	args := types.SecretArgs{}
	args.Name = p.PluginMeta.Name
	args.Namespace = p.PluginMeta.Namespace
	args.Type = p.Type
	args.Behavior = p.Behavior

	loader := p.h.Loader()
	validator := p.h.Validator()
	kvLoader := kv.NewLoader(loader, validator)

	files := p.GeneratorArgs.KvPairSources.FileSources

	for _, fileEntry := range files {
		key, fileName, err := parseFileEntry(fileEntry)
		if err != nil {
			return nil, sopsGenErr(err)
		}

		log.Debugf("Decrypting file '%s' as '%s'", fileName, key)

		cipherText, err := loader.Load(fileName)
		if err != nil {
			return nil, sopsGenErr(err)
		}

		clearText, err := decrypt.DataWithFormat(cipherText, formats.FormatForPath(fileName))
		if err != nil {
			return nil, sopsGenErr(err)
		}

		// Intuitively, this might look like it would breake because it could lead to
		// potentially broken input if this were a YAML or JSON file, but it is _not_. These are
		// just memory representations; it's not actually ever being _parsed_ as anything.
		// Internally the
		//
		// Ideally we could write our own KvLoader implementation and that returns a `[]kv.Pair`
		// And then use those in `Data`, but it would increase maintenance burden in case something
		// were to change with the KvLoader API, which is fairly stable, but it's still v1beta, so
		// until it reaches v1, I prefer this way.
		//
		// The k8s API itself handles encoding the values into Base64, so we don't have to do this here.
		args.LiteralSources = append(args.LiteralSources, key+"="+string(clearText))
	}

	envs := p.GeneratorArgs.KvPairSources.EnvSources
	envKvMap := make(kvMap)
	for _, envEntry := range envs {
		cipherText, err := loader.Load(envEntry)
		if err != nil {
			return nil, sopsGenErr(err)
		}

		log.Debugf("Decrypting env file '%s'", envEntry)

		clearText, err := decrypt.DataWithFormat(cipherText, formats.Dotenv)
		if err != nil {
			return nil, sopsGenErr(err)
		}

		err = parseDotEnvFile(clearText, validator, envKvMap)

		if err != nil {
			return nil, sopsGenErr(err)
		}
	}

	for name, value := range envKvMap {
		args.LiteralSources = append(args.LiteralSources, name+"="+value)
	}

	if log.GetLevel() == logrus.DebugLevel {
		keys := make([]string, 0, len(args.LiteralSources))

		for _, secret := range args.LiteralSources {
			name := strings.SplitN(secret, "=", 2)[0]
			keys = append(keys, name)
		}

		log.Debugf("Generating secret '%s' with %d entries: %v", args.Name, len(keys), strings.Join(keys, ", "))
	}

	return p.h.ResmapFactory().FromSecretArgs(kvLoader, args)
}

/*
Parses the file entry as: [key=]fileName

If no key is specified, it uses the basename of the filename as the key.
Loosely based on https://sigs.k8s.io/kustomize/api/kv/kv.go#L181-L204
*/
func parseFileEntry(source string) (key string, fileName string, err error) {
	parts := strings.Split(source, "=")

	switch len(parts) {
	case 1:
		return filepath.Base(source), source, nil
	case 2:
		key, fileName = parts[0], parts[1]
		if key == "" {
			return "", "", fmt.Errorf("key name for file %v missing, try removing the leading equal sign(=), or add a key name", fileName)
		}

		if fileName == "" {
			return "", "", fmt.Errorf("file path for key %v missing, try removing the trailing equal sign or add a file", key)
		}

		return key, fileName, nil
	default:
		return "", "", fmt.Errorf("file names or keys can't have equal signs (=), but %v was encountered which is ambiguous", source)
	}
}

/*
Parses ruby/node/docker style .env files. It allows empty env vars.
It doesn't support reading the env var from the OS when only the name is provided, since
SOPS considers this invalid syntax.
*/
func parseDotEnvFile(content []byte, validator ifc.Validator, envVars kvMap) error {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	// scanner.Split(bufio.ScanLines)

	lineNum := 0
	for scanner.Scan() {
		line := scanner.Bytes()

		// Strip UTF-8 byte order mark from first line
		if lineNum == 0 {
			line = bytes.TrimPrefix(line, utf8bom)
		}

		err := parseDotEnvLine(line, validator, envVars)
		if err != nil {
			return errors.Wrapf(err, "line %d", lineNum)
		}

		lineNum++
	}
	return scanner.Err()
}

func parseDotEnvLine(line []byte, validator ifc.Validator, envVars kvMap) error {
	if !utf8.Valid(line) {
		return fmt.Errorf("invalid UTF-8 bytes: %v", string(line))
	}

	line = bytes.TrimLeftFunc(line, unicode.IsSpace)

	// Empty line or comment
	if len(line) == 0 || line[0] == '#' {
		return nil
	}

	// We don't have to check for anything here since SOPS doesn't allow name-only entires
	// So this means we can't get read the value from the execution context's environemnt,
	// like the builtin SecretsGenerator does here https://sigs.k8s.io/kustomize/api/kv/kv.go#L173-L175
	// It's a neat idea, but I'd rather not introduce hacks or magic values to facilitate this unless
	// we really need it.
	kv := strings.SplitN(string(line), "=", 2)
	name := kv[0]
	value := kv[1]

	if err := validator.IsEnvVarName(name); err != nil {
		return err
	}

	envVars[name] = value

	return nil
}
