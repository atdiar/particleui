package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		showUsageAndExit()
	}

	switch os.Args[1] {
	case "precompile":
		handlePrecompileCommand(os.Args[2:])
	default:
		showUsageAndExit()
	}
}

func showUsageAndExit() {
	fmt.Println("Usage: gowasm <command> [arguments]")
	fmt.Println("Commands: precompile")
	os.Exit(1)
}

func handlePrecompileCommand(args []string) {
	precompileCmd := flag.NewFlagSet("precompile", flag.ExitOnError)
	stdLibFlag := precompileCmd.Bool("std", false, "Compile all standard library packages")
	packagePath := precompileCmd.String("package", "", "Specify a non-standard library package to compile")
	manifestFile := precompileCmd.String("manifest-file", "manifest.txt", "Manifest file name")

	precompileCmd.Parse(args)

	goroot := os.Getenv("GOROOT")
	gopath := os.Getenv("GOPATH")

	var packages []string
	if *stdLibFlag {
		packages = getStandardLibraryPackages(goroot)
	} else if *packagePath != "" {
		packages = []string{*packagePath}
		fetchPackage(*packagePath, gopath)
	} else {
		fmt.Println("No package specified.")
		os.Exit(1)
	}

	processed := make(map[string]bool)
	for _, pkg := range packages {
		err := processPackage(pkg, processed, goroot, gopath, *manifestFile)
		if err != nil {
			fmt.Printf("Error processing package %s: %v\n", pkg, err)
		}
	}
}

func getStandardLibraryPackages(goroot string) []string {
	cmd := exec.Command("go", "list", "std")
	if goroot != "" {
		cmd.Env = append(os.Environ(), "GOROOT="+goroot)
	}

	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error listing standard library packages: %v\n", err)
		os.Exit(1)
	}

	return strings.Fields(string(output))
}

func fetchPackage(packagePath, gopath string) {
	fmt.Printf("Fetching package: %s\n", packagePath)
	cmd := exec.Command("go", "get", packagePath)
	if gopath != "" {
		cmd.Env = append(os.Environ(), "GOPATH="+gopath)
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error fetching package %s: %v\n", packagePath, err)
	}
}

func processPackage(packagePath string, processed map[string]bool, goroot, gopath, manifestFile string) error {
	if processed[packagePath] {
		return nil
	}

	output, workDir, err := compileAndExtractWorkDir(packagePath, goroot, gopath)
	if err != nil {
		return fmt.Errorf("compilation error: %v\nOutput: %s", err, output)
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current directory: %v", err)
	}

	err = processCompiledPackages(workDir, dir, packagePath, manifestFile)
	if err != nil {
		return fmt.Errorf("error processing compiled packages: %v", err)
	}

	processed[packagePath] = true
	return nil
}

func compileAndExtractWorkDir(packagePath, goroot, gopath string) (string, string, error) {
	cmd := exec.Command("go", "install", "-work", "-a", packagePath)
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if goroot != "" {
		cmd.Env = append(cmd.Env, "GOROOT="+goroot)
	}
	if gopath != "" {
		cmd.Env = append(cmd.Env, "GOPATH="+gopath)
	}

	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	if err != nil {
		return output, "", err
	}

	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "WORK=") {
			workDir := strings.TrimPrefix(line, "WORK=")
			return output, workDir, nil
		}
	}

	return output, "", fmt.Errorf("work directory not found in output")
}

func processCompiledPackages(workDir, targetDir, packagePath, manifestFile string) error {
	importCfgPaths, err := filepath.Glob(filepath.Join(workDir, "*", "importcfg"))
	if err != nil {
		return err
	}

	for _, cfgPath := range importCfgPaths {
		err := processImportCfg(cfgPath, workDir, targetDir, manifestFile)
		if err != nil {
			return err
		}
	}

	mainPackageFile := filepath.Join(workDir, "b001", "_pkg_.a")
	targetMainPath := filepath.Join(targetDir, "pkg", packagePath+".a")
	return copyFile(mainPackageFile, targetMainPath, manifestFile)
}

func processImportCfg(cfgPath, workDir, targetDir, manifestFile string) error {
	file, err := os.Open(cfgPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), " ")
		if len(parts) == 2 && parts[0] == "packagefile" {
			name, path := strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
			targetPath := filepath.Join(targetDir, "pkg", name+".a")
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			if err := copyFile(path, targetPath, manifestFile); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst, manifestFile string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return updateManifest(manifestFile, src, dst)
}

func updateManifest(manifestFile, src, dst string) error {
	f, err := os.OpenFile(manifestFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s -> %s\n", src, dst))
	return err
}
