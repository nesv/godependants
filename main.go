package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("godependants: ")

	var (
		dir    string
		direct bool
		quiet  bool
	)
	flag.StringVar(&dir, "dir", "", "`directory` to run godependants in")
	flag.BoolVar(&direct, "direct", false, "only list direct dependencies (no transitive dependencies)")
	flag.BoolVar(&quiet, "quiet", false, "disable stderr output")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [packages...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if quiet {
		log.SetOutput(io.Discard)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pkgname, modname, err := currentPackage(ctx, dir)
	if err != nil {
		log.Fatalln("get module name:", err)
	}
	log.Println("module", modname)

	pkgs, err := loadModulePackages(ctx, dir)
	if err != nil {
		log.Fatalln("load module packages:", err)
	}

	pkgdeps := collectDependants(pkgs)
	trimExtModDeps(modname, pkgdeps)
	for pkg, dependants := range pkgdeps {
		log.Println(pkg, dependants)
	}

	if len(flag.Args()) == 0 {
		// Assume the current package is the one we would like to get
		// the dependants of.
		if direct {
			for _, name := range pkgdeps[pkgname] {
				fmt.Println(name)
			}
			return
		}

		for _, name := range dependantsOf(pkgname, pkgdeps) {
			fmt.Println(name)
		}
		return
	}

	toprint := make(map[string]struct{})
	for _, pkg := range flag.Args() {
		pkg, err := cleanPkgPath(pkg, modname)
		if err != nil {
			log.Println("clean package path:", err)
			continue
		}

		if _, ok := pkgdeps[pkg]; !ok {
			log.Println("no such package in module:", pkg)
			continue
		}

		if direct {
			for _, name := range pkgdeps[pkg] {
				fmt.Println(name)
			}
			return
		}

		for _, name := range dependantsOf(pkg, pkgdeps) {
			toprint[name] = struct{}{}
		}
	}
	for name := range toprint {
		fmt.Println(name)
	}
}

// currentPackage returns the name of the package and module.
func currentPackage(ctx context.Context, dir string) (packageName, moduleName string, err error) {
	cfg := packages.Config{
		Mode:    packages.NeedName | packages.NeedModule,
		Context: ctx,
		Dir:     dir,
	}
	pkgs, err := packages.Load(&cfg, "")
	if err != nil {
		return "", "", fmt.Errorf("load current package: %w", err)
	}
	if len(pkgs) != 1 {
		return "", "", fmt.Errorf("wrong number of packages loaded: wanted %d, got %d", 1, len(pkgs))
	}

	if pkgs[0].Module == nil {
		return "", "", errors.New("current package is not located within a module")
	}

	return pkgs[0].ID, pkgs[0].Module.Path, nil
}

func loadModulePackages(ctx context.Context, dir string) ([]*packages.Package, error) {
	cfg := packages.Config{
		Mode:    packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedModule,
		Context: ctx,
		Dir:     dir,
	}
	pkgs, err := packages.Load(&cfg, filepath.Join(".", "..."))
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	return pkgs, nil
}

func collectDependants(pkgs []*packages.Package) map[string][]string {
	deps := make(map[string][]string)
	packages.Visit(pkgs, func(p *packages.Package) bool {
		if !packageIsExternal(p.ID) {
			return false
		}
		// fmt.Println("::", p.ID)

		for name := range p.Imports {
			if !packageIsExternal(name) {
				continue
			}

			pp, ok := deps[name]
			if !ok {
				pp = []string{}
			}
			pp = append(pp, p.ID)
			deps[name] = pp
		}

		return true
	}, nil)
	return deps
}

// packageIsExternal indicates if the given package name is an external
// package (i.e. not in the standard library).
func packageIsExternal(name string) bool {
	i := strings.Index(name, "/")
	if i == -1 {
		return false
	}

	return strings.Contains(name[:i], ".")
}

// trimExtModDeps removes all packages (keys) from m, that do not have a
// dependency on them from the local module.
func trimExtModDeps(localModName string, m map[string][]string) {
	var rm []string
	for pkg, dependants := range m {
		var hasLocalModDep bool
		for _, dep := range dependants {
			if strings.HasPrefix(dep, localModName) {
				hasLocalModDep = true
			}
		}
		if !hasLocalModDep {
			rm = append(rm, pkg)
		}
	}
	for _, pkg := range rm {
		delete(m, pkg)
	}
}

func dependantsOf(pkgname string, m map[string][]string) []string {
	log.Println("dependants of", pkgname)
	deps, ok := m[pkgname]
	if !ok {
		return nil
	}

	dm := make(map[string]struct{})
	for _, name := range deps {
		dm[name] = struct{}{}
		log.Println("adding dependant", name)
		for _, transitive := range dependantsOf(name, m) {
			dm[transitive] = struct{}{}
			log.Println("adding transitive dependant", transitive)
		}
	}

	var dependants []string
	for name := range dm {
		dependants = append(dependants, name)
	}
	return dependants
}

func cleanPkgPath(name, modName string) (string, error) {
	if strings.HasPrefix(name, "./") {
		name = filepath.Join(modName, name[2:])
	}

	log.Println("cleaned:", name)

	return name, nil
}
