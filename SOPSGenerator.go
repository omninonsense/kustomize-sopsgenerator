package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/decrypt"

	logrus "github.com/sirupsen/logrus"
	sopsLogging "go.mozilla.org/sops/v3/logging"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/ifc"
	"sigs.k8s.io/kustomize/api/kv"
	"sigs.k8s.io/kustomize/api/loader"
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

//nolint: unused,deadcode
var KustomizePlugin plugin

// We use the SOPS wrapped logrus, so we can share loglevels with them
var log = sopsLogging.NewLogger(Kind)

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

/*
SOPSGenerator reads SOPS encrypted files from the disk, saves the cleartext versions into an in-memory filesystem, and then
chains to the builtin  SecretsGenerator with an in-memory loader.
*/
func (p *plugin) Generate() (resmap.ResMap, error) {
	args := types.SecretArgs{}
	args.Name = p.PluginMeta.Name
	args.Namespace = p.PluginMeta.Namespace
	args.Type = p.Type
	args.Behavior = p.Behavior
	args.KvPairSources = p.GeneratorArgs.KvPairSources

	pluginLoader := p.h.Loader()
	validator := p.h.Validator()

	memfs := filesys.MakeFsInMemory()

	// TODO: Would be nice to match the restriction from the "parent" loader, but I have no idea where to read the restriction from?
	// We don't really _need_ to have any restrictions on the in-memory loader, since "our" loader will respect the restrictions
	secGenLoader, err := loader.NewLoader(loader.RestrictionNone, filesys.SelfDir, memfs)
	secGenKvLoader := kv.NewLoader(secGenLoader, validator)

	if err != nil {
		log.Panicf("Error creating new loader: %v", err)
	}

	files := p.GeneratorArgs.KvPairSources.FileSources
	for _, fileEntry := range files {
		_, file, err := parseFileEntry(fileEntry)

		if err != nil {
			return nil, sopsGenErr(err)
		}

		err = decryptFileIntoFs(file, pluginLoader, memfs)
		if err != nil {
			return nil, sopsGenErr(err)
		}
	}

	envs := p.GeneratorArgs.KvPairSources.EnvSources
	for _, envEntry := range envs {
		err = decryptFileIntoFs(envEntry, pluginLoader, memfs)
		if err != nil {
			return nil, sopsGenErr(err)
		}
	}

	return p.h.ResmapFactory().FromSecretArgs(secGenKvLoader, args)
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
Loads a file using the specified Loader (with all restrictions applied), decrypts it using SOPS, and then
writes it into the provided filesys.
*/
func decryptFileIntoFs(file string, loader ifc.Loader, fs filesys.FileSystem) (err error) {
	log.Infof("Decrypting file '%s'", file)

	cipherText, err := loader.Load(file)
	if err != nil {
		return err
	}

	clearText, err := decrypt.DataWithFormat(cipherText, formats.FormatForPath(file))
	if err != nil {
		return err
	}

	err = fs.WriteFile(file, clearText)

	return err
}
