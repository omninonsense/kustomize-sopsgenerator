package main

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/konfig"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, "You need to provide a subcommand!\n")
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "kind":
		fmt.Println(Kind)
	case "subdir":
		fmt.Println(subdir())
	case "plugin-home":
		fmt.Println(pluginHome())
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand %s\n", cmd)
		os.Exit(1)
	}
}

func subdir() string {
	return fmt.Sprintf("%s/%s/%s", Domain, Version, strings.ToLower(Kind))
}

func pluginHome() filesys.ConfirmedDir {
	fs := filesys.MakeFsOnDisk()

	ph, err := konfig.DefaultAbsPluginHome(fs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Kustomize error: %s\n", err)
		fmt.Fprintf(os.Stderr, "Create one of the above mentioned standard directories, or set $%s\n", konfig.KustomizePluginHomeEnv)
		os.Exit(1)
	}

	dir, _, err := fs.CleanedAbs(ph)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	return dir
}
