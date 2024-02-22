package main

import (
	"bufio"
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
		panic(err)
	}
	

	// Incorporate Go version into compiledDir path
	compiledDir, err := filepath.Abs("./lib_prec_" + version)
	if err != nil {
		panic(err)
	}

	// Load or initialize importmap
	importcfgpath := filepath.Join(compiledDir, "importcfg")
	importmap, err := loadImportMap(importcfgpath)
	if err != nil {
		panic(err)
	}

	if importmap == nil {
		importmap = make(map[string]string)
		// we need to compile the standard library

		// Compile the standard library
		if err := compileStandardLibrary(compiledDir); err != nil {
			panic(err)
		}

		if err = generateImportCfg(compiledDir, &importmap); err != nil {
			panic(err)
		}
	}

	if library != "" {
		if err := compileNonStdImports(library, compiledDir, importmap); err != nil {
			panic(err)
		}
	}

	// Let's compilte the toolchain for wasm
	err = compileGoToolForWASM("compile", compiledDir)
	if err != nil {
		fmt.Errorf("failed to compile go tool compile: %v", err)
		os.Exit(1)
	}

	err = compileGoToolForWASM("link", compiledDir)
	if err != nil {
		fmt.Errorf("failed to compile go tool link: %v", err)
		os.Exit(1)
	}

	err = compileGoToolForWASM("gofmt", compiledDir)
	if err != nil {
		fmt.Errorf("failed to compile go tool fmt: %v", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Compilation complete, files are in: %s\n", compiledDir)
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

	fmt.Println("Successfully compiled stdlib to", targetDir)
	return nil
}

// compileNonStdImports compiles a given library and its transtitive dependencies.
func compileNonStdImports(library, targetDir string, importmap map[string]string) error {
	// Compile your library and its dependencies
	if verbose {
		fmt.Println("Compiling library and its dependencies...")
	}

	// Get dependencies of the library
	deps, err := getDependencies(library)
	if err != nil {
		return err
	}

	// Open the importcfg file for modification, appending at the end, or create it if it doesn't exist
	importcfg, err := os.OpenFile(importcfgpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open importcfg file: %v", err)
	}
	defer importcfg.Close()

	// Compile each package in the dependency list
	for _, pkg := range deps {
		if _, exists := importmap[pkg]; !exists { // Check if already compiled
			if err := compilePackage(pkg, targetDir, importmap); err != nil {
				fmt.Printf("Failed to compile %s: %v\n", pkg, err)
			}
			// append to importcfg
			importPath := strings.Replace(pkg, filepath.Dir(targetDir)+string(os.PathSeparator), "", 1)
			importPath = strings.Replace(importPath, string(os.PathSeparator), "/", -1)
			entry := fmt.Sprintf("packagefile %s=%s\n", importPath, importmap[pkg])
			if verbose {
				fmt.Printf("Adding to importcfg: %s", entry)
			}
			_, err = importcfg.WriteString(entry)
			if err != nil {
				return fmt.Errorf("failed to write to importcfg file: %v", err)
			}

		}
	}
	return nil
}

func getDependencies(pkg string) ([]string, error) {
	cmd := exec.Command("go", "list", "-deps", pkg)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(stdout)), "\n"), nil
}

func compilePackage(pkg string, targetDir string, importmap map[string]string) error {
	// Find source files for the package
	srcFiles, err := getSourceFiles(pkg)
	if err != nil {
		return err
	}

	if len(srcFiles) == 0 {
		return nil // No source files, skip package
	}

	baseDir := filepath.Base(targetDir)

	// Compile the package
	pkgDir := filepath.Join(targetDir, pkg)
	err = os.MkdirAll(pkgDir, 0777)
	if err != nil {
		return err
	}

	output := filepath.Join(pkgDir, filepath.Base(pkg)+".a")
	args := append([]string{"tool", "compile", "-std", "-o", output, "-p", pkg}, srcFiles...)
	cmd := exec.Command("go", args...)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Update importmap upon successful compilation
	importmap[pkg] = filepath.Join(baseDir,output)

	if verbose {
		fmt.Printf("Compiled %s\n", pkg)
	}

	return nil
}

func getSourceFiles(pkg string) ([]string, error) {
	// 'go list -f "{{.GoFiles}}" pkg' to get source files
	cmd := exec.Command("go", "list", "-f", "{{range .GoFiles}}{{$.Dir}}/{{.}} {{end}}", pkg)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	files := strings.Fields(strings.TrimSpace(string(stdout)))
	return files, nil
}

func defaultGOROOT() string {
	cmd := exec.Command("go", "env", "GOROOT")
	stdout, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(stdout))
}

// Walks through the directory to find all .a files and maps them.
func generateImportMap(targetDir string) (map[string]string, error) {
	importMap := make(map[string]string)
	baseDir:= filepath.Base(targetDir)

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
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
			importMap[importPath] = filepath.Join(baseDir, relPath)
		}
		return nil
	})

	return importMap, err
}

// Generates an importcfg file from the .a files in targetDir.
func generateImportCfg(targetDir string, pImportmap *map[string]string) error {
	importMap, err := generateImportMap(targetDir)
	if err != nil {
		return fmt.Errorf("error generating import map: %v", err)
	}
	imap := *pImportmap
	for k, v := range importMap {
		imap[k] = v
	}

	// Write the importcfg file
	importcfgPath := filepath.Join(targetDir, "importcfg")
	file, err := os.Create(importcfgPath)
	if err != nil {
		return fmt.Errorf("failed to create importcfg file: %v", err)
	}
	defer file.Close()

	for importPath, relPath := range importMap {
		entry := fmt.Sprintf("packagefile %s=%s\n", importPath, relPath)
		_, err = file.WriteString(entry)
		if err != nil {
			return fmt.Errorf("failed to write to importcfg file: %v", err)
		}
	}

	fmt.Println("importcfg file generated successfully at:", importcfgPath)
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

	cmd := exec.Command("go", "build", "-o", filepath.Join(targetDir,fmt.Sprintf("%s.wasm", toolName)), ".")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Dir = toolPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to compile %s: %s\n", toolName, string(output))
		fmt.Print(err)
		return err
	}

	fmt.Printf("%s compiled successfully to %s.wasm\n", toolName, toolName)
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

/*
func main() {
    // Adjust these paths as necessary for your setup
    err := GenerateImportCfgLink("main.go", "importcfg", "importcfg.link")
    if err != nil {
        fmt.Printf("Error generating importcfg.link: %v\n", err)
    } else {
        fmt.Println("Successfully generated importcfg.link")
    }
}
*/
