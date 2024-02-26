package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"

	//"io"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	library string
	verbose bool
	debug   bool
	v       *bool
	d       *bool

	importcfgpath string
	DefaultDeps   = []string{
		"github.com/atdiar/particleui",
		"github.com/atdiar/particleui/drivers/js/compat",
		"github.com/atdiar/particleui/drivers/js",
	}
)

func init() {
	flag.StringVar(&library, "library", library, "Library to compile")
	flag.BoolVar(&verbose, "verbose", verbose, "Verbose output")
	flag.BoolVar(&debug, "debug", debug, "Debug output")
	d = flag.Bool("d", false, "Debug output")
	v = flag.Bool("v", false, "Verbose output")

}

func main() {

	flag.Parse()

	if *v {
		verbose = true
	}

	version, err := getGoVersion()
	if err != nil {
		fmt.Printf("failed to get Go version: %v", err)
		os.Exit(1)
	}

	// everything should be put in the wasmgc directory created from the current working directory
	err = os.MkdirAll("wasmgc", 0755)
	if err != nil {
		fmt.Printf("Unable to create the destination directory: %v \n", err)
		os.Exit(1)
	}

	dir := filepath.Join(".", "wasmgc")

	// Incorporate Go version into compiledDir path
	libDir := filepath.Join(dir, "library")

	// Load or initialize importmap
	importcfgpath = filepath.Join(dir, "importcfg")
	importmap, err := loadImportMap(importcfgpath)
	if err != nil {
		fmt.Printf("failed to load importcfg: %v", err)
		os.Exit(1)
	}

	if importmap == nil {
		importmap = make(map[string]string)
		// we need to compile the standard library

		// Compile the standard library
		if err = compileStandardLibrary(libDir); err != nil {
			fmt.Printf("failed to compile standard library: %v", err)
			os.Exit(1)
		}

		// Create the importcfg file
		if err = generateImportCfg(dir, libDir, &importmap); err != nil {
			fmt.Printf("failed to create importcfg: %v", err)
			os.Exit(1)
		}
	}

	// TODO make sure that the compilation occurs with build tag !server
	for _, dep := range DefaultDeps {
		if err = compileNonStandardPkg(dep, libDir); err != nil {
			fmt.Printf("failed to compile %s: %v", dep, err)
			os.Exit(1)
		}
	}

	if library != "" {
		if err := compileNonStandardPkg(library, libDir); err != nil {
			fmt.Printf("failed to compile %s: %v", library, err)
			os.Exit(1)
		}
	}

	// Update the importcfg file
	if err = generateImportCfg(dir, libDir, &importmap); err != nil {
		fmt.Printf("failed to create importcfg: %v", err)
		os.Exit(1)
	}

	// Let's compile the toolchain for wasm
	err = compileGoToolForWASM("compile", dir)
	if err != nil {
		fmt.Printf("failed to compile go tool compile: %v", err)
		os.Exit(1)
	}

	err = compileGoToolForWASM("link", dir)
	if err != nil {
		fmt.Printf("failed to compile go tool link: %v", err)
		os.Exit(1)
	}

	err = compileGoToolForWASM("gofmt", dir)
	if err != nil {
		fmt.Printf("failed to compile go tool fmt: %v", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Compilation complete, files are in: %s\n", dir)
	}

	// Now we should generate the prefetchlist file which is equivalent to the
	// These will essentially be the list of dependencies that the browser will start fetching by default
	// and store in indexedDB if not already in there.
	// This is a way to reduce the time it takes to load the application.
	var prefetchList = make(map[string]string)
	for _, dep := range DefaultDeps {
		d, err := getDependencies(dep)
		if err != nil {
			fmt.Printf("Unable to generate prefetch list: failed to get dependencies for %s: %v", dep, err)
			os.Exit(1)
		}
		for _, pkg := range d {
			pkgpath, ok := importmap[pkg]
			if !ok {
				panic("The package " + pkg + " is not in the importmap in spite of being a dependency of the compiled package" + dep)
			}
			prefetchList[pkg] = pkgpath
		}
	}
	// Write the prefetch list to a json file
	prefetchListPath := filepath.Join(dir, "prefetchlist.json")
	file, err := os.Create(prefetchListPath)
	if err != nil {
		fmt.Printf("Unable to create prefetchlist file: %v", err)
		os.Exit(1)
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(prefetchList)
	if err != nil {
		fmt.Printf("Unable to encode prefetchlist: %v", err)
		os.Exit(1)
	}

	// let's create manifest.json
	manifest := map[string]string{
		"goversion":    version,
		"importcfg":    filepath.Join(dir, "importcfg"),
		"prefetchlist": prefetchListPath,
		"compile":      filepath.Join(dir, "compile.wasm"),
		"link":         filepath.Join(dir, "link.wasm"),
		"gofmt":        filepath.Join(dir, "gofmt.wasm"),
		"libraries":    libDir,
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	file, err = os.Create(manifestPath)
	if err != nil {
		fmt.Printf("Unable to create manifest file: %v", err)
		os.Exit(1)
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(manifest)
	if err != nil {
		fmt.Printf("Unable to encode manifest: %v", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Manifest file created at: %s\n", manifestPath)
	}

	if verbose {
		fmt.Println("SUCCESS!")
	}

}

func compileStandardLibrary(targetDir string) error {
	if verbose {
		fmt.Println("Compiling standard library...")
	}

	// Use go install with the -pkgdir flag to specify the output directory for .a files
	args := []string{"install", "-a"}
	if verbose {
		args = append(args, "-v")
	}

	if debug {
		//args = append(args, "-n")
		args = append(args, "-work")
	}
	args = append(args, "-pkgdir", targetDir, "std")

	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOARCH=wasm", "GOOS=js", "GODEBUG=installgoroot=all")

	// Capture output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error compiling stdlib:", err)
		return err
	}
	if verbose {
		fmt.Println("Successfully compiled stdlib to", targetDir)
	}
	return nil
}

func compileNonStandardPkg(library, targetDir string) error {
	if verbose {
		fmt.Printf("Compiling %s and its dependencies...\n", library)
	}

	// Transform the library import path into a directory structure
	libraryPath := filepath.Join(targetDir, filepath.FromSlash(library))
	// Ensure that the directory structure exists
	if err := os.MkdirAll(libraryPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory structure for %s: %v", library, err)
	}

	args := []string{"install", "-a"}
	if verbose {
		args = append(args, "-v")
	}

	if debug {
		//args = append(args, "-n")
		args = append(args, "-work")
	}
	 // filepath.Dir(libraryPath) // ??
	args = append(args, "-tags", "!server", "-pkgdir", targetDir, library)

	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOARCH=wasm", "GOOS=js", "GODEBUG=installgoroot=all")
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to compile %s: %v", library, err)
	}

	if verbose {
		fmt.Printf("Successfully compiled %s to %s\n", library, libraryPath)
	}
	return nil
}

func getDependencies(pkg string) ([]string, error) {
	cmd := exec.Command("go", "list", "-deps", pkg)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// DEBUG
	fmt.Println("the list of dependencies for", pkg, "is", string(stdout))

	return strings.Split(strings.TrimSpace(string(stdout)), "\n"), nil
}

// Walks through the directory to find all .a files and maps them.
func generateImportMap(baseDir, targetDir string) (map[string]string, error) {
	importMap := make(map[string]string)
	targetDirPath, err := filepath.Rel(baseDir, targetDir)
	if err != nil {
		return importMap, err
	}

	err = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".a" {
			// Generate the import path key
			relPath, err := filepath.Rel(targetDir, path)
			if err != nil {
				return err
			}
			importPath := strings.TrimSuffix(relPath, ".a")
			importPath = strings.ReplaceAll(importPath, string(filepath.Separator), "/")
			importMap[importPath] = filepath.Join(filepath.Base(baseDir), targetDirPath, relPath)
		}
		return nil
	})

	return importMap, err
}

// Generates an importcfg file from the .a files in targetDir.
func generateImportCfg(baseDir, targetDir string, pImportmap *map[string]string) error {
	importMap, err := generateImportMap(baseDir, targetDir)
	if err != nil {
		return fmt.Errorf("error generating import map: %v", err)
	}
	imap := *pImportmap
	for k, v := range importMap {
		imap[k] = v
	}

	file, err := os.Create(importcfgpath)
	if err != nil {
		return fmt.Errorf("failed to open importcfg file: %v", err)
	}
	defer file.Close()

	for importPath, relPath := range importMap {
		entry := fmt.Sprintf("packagefile %s=%s\n", importPath, relPath)
		_, err = file.WriteString(entry)
		if err != nil {
			return fmt.Errorf("failed to write to importcfg file: %v", err)
		}
	}

	if verbose {
		fmt.Println("importcfg file generated successfully at:", importcfgpath)
	}

	return nil
}

func loadImportMap(importcfgpath string) (map[string]string, error) {
	var importmap map[string]string

	// Check if importcfg exists
	if _, err := os.Stat(importcfgpath); os.IsNotExist(err) {
		return importmap, nil // No existing importcfg, return empty map
	}

	// Load existing importcfg
	content, err := os.ReadFile(importcfgpath)
	if err != nil {
		return importmap, err
	}

	importmap = make(map[string]string)

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "packagefile ") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				pkgPath := parts[0][12:] // Remove "packagefile " prefix
				binaryPath := parts[1]
				importmap[pkgPath] = binaryPath
			}
		}
	}

	return importmap, nil
}

func getGoVersion() (string, error) {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Example output: go version go1.15.2 linux/amd64
	// We extract and return the version part: "go1.15.2"
	fields := strings.Fields(string(output))
	if len(fields) < 3 {
		return "", errors.New("unexpected go version format")
	}

	version := fields[2] // The version part
	return version, nil
}

func compileGoToolForWASM(toolName string, targetDir string) error {
	goroot := runtime.GOROOT()
	toolPath := filepath.Join(goroot, "src", "cmd", toolName)
	abstargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join(abstargetDir, fmt.Sprintf("%s.wasm", toolName)), ".")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Dir = toolPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to compile %s: %s\n", toolName, string(output))
		fmt.Print(err)
		return err
	}

	if verbose {
		fmt.Printf("%s compiled successfully to %s.wasm\n", toolName, toolName)

	}
	return nil
}

// GenerateImportCfgLink generates the importcfg.link file.
func GenerateImportCfgLink(mainFilePath, importcfgpath, outputFilePath string) error {
	imports, err := ExtractImports(mainFilePath)
	if err != nil {
		return err
	}

	importCfg, err := os.ReadFile(importcfgpath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(importCfg), "\n")
	importMap := make(map[string]string)
	for _, line := range lines {
		parts := strings.Split(line, "=")
		if len(parts) == 2 {
			importMap[parts[0][12:]] = parts[1] // Remove "packagefile " prefix
		}
	}

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Write the special first line for the main package
	_, err = outputFile.WriteString("packagefile command-line-arguments=main.a\n")
	if err != nil {
		return err
	}

	// Write the dependencies
	for _, imp := range imports {
		if path, exists := importMap[imp]; exists {
			_, err := outputFile.WriteString(fmt.Sprintf("packagefile %s=%s\n", imp, path))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ExtractImports parses the main.go file and extracts import paths.
func ExtractImports(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var imports []string
	scanner := bufio.NewScanner(file)
	inImportBlock := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "import (") {
			inImportBlock = true
			continue
		} else if inImportBlock && strings.Contains(line, ")") {
			break
		}

		if inImportBlock || strings.HasPrefix(strings.TrimSpace(line), "import ") {
			trimmed := strings.Trim(line, "\t \"")
			if trimmed != "" && trimmed != "import" {
				imports = append(imports, trimmed)
			}
		}
	}

	return imports, scanner.Err()
}
