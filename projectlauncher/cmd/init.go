/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"io"
	"strings"
	"os"
	"os/exec"
	"encoding/json"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var verbose bool

var interactive, graphic bool
var projectName string
var web, mobile, desktop, terminal string

var config map[string]string
const configFileName = "zui_config.json"

func configExists() bool {
	_, err := os.Stat(configFileName)
	return !os.IsNotExist(err)
}

func LoadConfig() error {
	file, err := os.Open(configFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return err
	}
	return nil
}

func SaveConfig() error {
	file, err := os.Create(configFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Setting indentation for formatted output
	return encoder.Encode(config)
}


// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init command is used to launch a new GUI project",
	Long: `
		init is the initialization command for a new GUI project.
		It creates the project structure and the configuration files.
		The project should be named, typically by providing the
		URL of the project repository
		It accepts the platform as mandatory argument, among which are:
		- web
		- mobile
		- desktop
		- terminal
		Each platform has different possible build targets, which are:
		- web: 
			o csr (client-side rendering)
			o ssr (server-side rendering)
			o ssg (static site generation)
		- mobile:
			o android
			o ios
		- desktop:
			o windows
			o linux
			o darwin (macos)
		- terminal: (in general, these are built cross-platform and are not specified until build time)
			o windows
			o linux
			o macos (darwin)
		
		Such an initialized project may only target one platform.
		To target multiplatform, you should create a project for each platform.
	`,
	Example: `
		zui init github.com/stephenstrange/kamartajwebapp -web
		zui init github.com/stephenstrange/kamartajiosapp -mobile=android
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if interactive {
			// Run the interactive mode function
			runInteractiveMode()
			return
		}

		if graphic {
			// Run the graphic mode function
			runGraphicMode()
			return
		}

		if len(args) > 0 {
			projectName = args[0]
		} else {
			fmt.Println("Error: Project name is required.")
			return
		}
	
		platformsSpecified := 0
	
		if web != "" {
			platformsSpecified++
		}
		if mobile != "" {
			platformsSpecified++
		}
		if desktop != "" {
			platformsSpecified++
		}
		if terminal != "" {
			platformsSpecified++
		}
	
		if platformsSpecified > 1 {
			fmt.Println("Error: Please specify only one platform (web, mobile, desktop, terminal).")
			os.Exit(1)
			return
		}

		// TODO: Check that the project config file is valid i.e. the initialization has been correctly done.

		if configExists() {
			fmt.Println("Error: A project already exists in this directory.")
			os.Exit(1)
			return
		}

		// git should ignore the release directory
		createFile(".gitignore", "/release")

		if web != "" {
			// handle web project initialization
			config["projectName"] = projectName
			config["platform"] = "web"
			

			// project initialization logic
			// should create the directories, basic template file, dev directory with dev server runnable source

			// Create dev directory if it doesn't already exists
			err:= createFolder(filepath.Join(".","dev"))
			if err!= nil{
				fmt.Println("Error: Could not create dev directory.")
				os.Exit(1)
				return
			}

			// dev holds the source code for the app.
			//
			// zui build compiles in CSR mode by default.
			// It also builds the server executable in dev/server/csr.
			//
			// zui build --ssr compiles in CSR mode and
			// puts the code for the CSR server in dev/server/ssr
			// the index.html file in dev/bin won't be served by the dev server.
			//
			// zui build --ssg compiles in SSG mode and output the file in dev/build/ssg.
			//
			// zui run -dev starts the dev server in CSR mode by default.
			// It serves the index.html file in dev/build as well as the
			// compiled app and the assets.
			//
			// zui run -dev -ssr starts the dev server in SSR mode.
			//
			// zui run -dev -ssg starts the dev server in SSG mode.
			// It serves the dev/build/ssg directory.
			//
			// -port might be an option for the dev server.
			//

			// Default build: on project initialization, a default project is 
			// created in the dev directory.
			// A sort of hello world app that can be run with zui run -dev.
			//
			// In the future, it should be possible to run zui init -template= template_URL
			// to create a project from a template. (TODO: use go new)

			// Let's create the default main.go file in the dev directory.
			// This will contain a default app that outputs a hello world, a game or something.
			// The default app should be a module, so run go mod init in the current directory.
			// The module name should be the project name.

			
			
			err = createFolder(filepath.Join(".","dev","build"))
			if err!= nil{
				fmt.Println("Error: Could not create dev/build directory.")
				os.Exit(1)
				return
			}

			err = createFolder(filepath.Join(".","dev","build","app"))
			if err!= nil{
				fmt.Println("Error: Could not create dev/build/app directory.")
				os.Exit(1)
				return
			}

			// Default main.go file
			err = createFile(filepath.Join(".","dev","build","app", "main.go"), defaultprojectfile)
			if err!= nil{
				fmt.Println("Error: Could not create dev/build/app/main.go file.")
				os.Exit(1)
				return
			}

			if verbose{
				fmt.Println("default main.go file created.")
			}

			// Default index.html file
			err = createFile(filepath.Join(".","dev","build","app","index.html"), defaultindexfile)
			if err!= nil{
				fmt.Println("Error: Could not create dev/build/app/index.html file.")
				os.Exit(1)
				return
			}

			if verbose{
				fmt.Println("default index.html file created.")
			}

			// copy wasm_exec.js to the ./dev/build/app directory
			err = CopyWasmExecJs(filepath.Join(".","dev","build","app"))
			if err!= nil{
				fmt.Println("Error: Could not copy wasm_exec.js file.")
				os.Exit(1)
				return
			}

			if verbose{
				fmt.Println("wasm_exec.js file copied from Go distribution.")
			}

			// This should be a module, so run go mod init in the current directory.
			// The module name should be the project name.
			err = initGoModule(projectName)
			if err!= nil{
				fmt.Println("Error: Could not initialize go module.")
				os.Exit(1)
				return
			}

			if verbose{
				fmt.Println("go module initialized.")
			}

			// Let's build the default app.
			// The output file should be in dev/build/app/main.wasm
			err = Build(filepath.Join(".","dev","build","app", "main.wasm"))
			if err != nil {
				fmt.Println("Error: Could not build the default app.")
				os.Exit(1)
				return
			}

			if verbose{
				fmt.Println("default app built.")
			}

			// Let's build the default server.
			// The output file should be in dev/build/server/csr/
			err = Build(filepath.Join(".","dev","build","server", "csr","main"))
			if err != nil {
				fmt.Println("Error: Could not build the default server.")
				os.Exit(1)
				return
			}

			if verbose{
				fmt.Println("default server built.")
			}

			// Config file should be valid now.
			if err := SaveConfig(); err != nil {
				fmt.Println("Error: Could not save configuration file.")
				os.Exit(1)
				return
			}
			if verbose{
				fmt.Println("SUCCESS! Your project has been initialized.")
			}




			// Process webOptions further
		} else if mobile != "" {
			// handle mobile initialization
			mobileOptions := strings.Split(mobile, ",")
			validMobileOptions := map[string]bool{"android": true, "ios": true}
			for _, option := range mobileOptions {
				if !validMobileOptions[option] {
					fmt.Printf("Error: Invalid mobile option '%s'\n", option)
					return
				}
			}
			// Process mobileOptions further
			fmt.Println("Mobile platform not yet implemented.")
		} else if desktop != "" {
			// handle desktop initialization
			desktopOptions := strings.Split(desktop, ",")
			validDesktopOptions := map[string]bool{"windows": true, "linux": true, "darwin": true}
			for _, option := range desktopOptions {
				if !validDesktopOptions[option] {
					fmt.Printf("Error: Invalid desktop option '%s'\n", option)
					return
				}
			}
			// Process desktopOptions further
			fmt.Println("Desktop platform not yet implemented.")
		} else if terminal != "" {
			// handle terminal initialization
			config["projectName"] = projectName
			config["platform"] = "terminal"
			if err := SaveConfig(); err != nil {
				fmt.Println("Error: Could not save configuration file.")
				return
			}
		} else {
			fmt.Println("Error: A platform (web, mobile, desktop, terminal) must be specified.")
		}
	},
	
	
}

func createFolder(path string) error{
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}
	if verbose {
		fmt.Printf("%s directory created.\n", path)
	}
	return nil
}

func createFile(path, content string) error{
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Write the content to the file
		_, err = file.WriteString(content)
		if err != nil {
			return err
		}
		if verbose {
			fmt.Printf("%s file created.\n", path)
		}
		return nil
	}
	if verbose {
		fmt.Printf("%s file already exists.\n", path)
	}
	return nil
}

// CopyWasmExecJs copies the wasm_exec.js file from the Go distribution to the specified destination directory.
func CopyWasmExecJs(destinationDir string) error {
	// Determine the Go root directory
	goRoot := os.Getenv("GOROOT") // GOROOT environment variable points to the Go installation
	if goRoot == "" {
		return fmt.Errorf("GOROOT environment variable is not set")
	}

	// Source wasm_exec.js path
	source := filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js")

	// Ensure the destination directory exists
	err := os.MkdirAll(destinationDir, os.ModePerm) // Create the destination directory if it does not exist
	if err != nil {
		return fmt.Errorf("error creating destination directory: %v", err)
	}

	// Destination wasm_exec.js path
	destination := filepath.Join(destinationDir, "wasm_exec.js")

	// Copy the wasm_exec.js file
	err = copyFile(source, destination)
	if err != nil {
		return fmt.Errorf("error copying file: %v", err)
	}

	return nil
}

func initGoModule(moduleName string) error {
	cmd := exec.Command("go", "mod", "init", moduleName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error initializing go module: %s, output: %s", err, output)
	}
	fmt.Printf("Successfully initialized go module: %s\n", moduleName)
	return nil
}

// copyFile is a helper function that copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

func Build(outputPath string, buildTags ...string) error {
	// Check if the build is for WebAssembly and save the current environment
	isWasm := strings.HasSuffix(outputPath, ".wasm")
	var originalGOOS, originalGOARCH string
	if isWasm {
		originalGOOS = os.Getenv("GOOS")
		originalGOARCH = os.Getenv("GOARCH")
		os.Setenv("GOOS", "js")
		os.Setenv("GOARCH", "wasm")
	}

	// Defer the restoration of the environment variables
	defer func() {
		if isWasm {
			os.Setenv("GOOS", originalGOOS)
			os.Setenv("GOARCH", originalGOARCH)
		}
	}()

	// Determine the correct file extension for the executable for non-WASM builds
	if !isWasm {
		goos := os.Getenv("GOOS")
		if goos == "" {
			goos = runtime.GOOS // Default to the current system's OS if GOOS is not set
		}
		if goos == "windows" && !strings.HasSuffix(outputPath, ".exe") {
			outputPath += ".exe"
		}
	}

	// Ensure the output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	args := []string{"build"}

	// Add build tags if provided
	if len(buildTags) > 0 {
		args = append(args, "-tags", strings.Join(buildTags, ","))
	}

	// Set the output file
	args = append(args, "-o", outputPath)

	// Specify the source file
	sourceFile := filepath.Join("dev", "main.go")
	args = append(args, sourceFile)

	// Execute the build command
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("build failed: %v", err)
	}
	
	return nil
}


func runInteractiveMode() {
	var input string

	// Prompt for project name
	fmt.Print("Project name: ")
	fmt.Scanln(&input)
	projectName = input

	// Prompt for platform
	fmt.Print("Choose a platform (web/mobile/desktop/terminal): ")
	fmt.Scanln(&input)

	switch input {
	case "web":
		fmt.Print("Choose a target for web (csr/ssr/ssg): ")
		fmt.Scanln(&input)
		web = input
	case "mobile":
		fmt.Print("Choose a target for mobile (android/ios): ")
		fmt.Scanln(&input)
		mobile = input
	case "desktop":
		fmt.Print("Choose a target for desktop (windows/darwin/linux): ")
		fmt.Scanln(&input)
		desktop = input
	case "terminal":
		terminal = "terminal" // default for terminal
	default:
		fmt.Println("Invalid platform selected.")
		return
	}

	// Continue with the rest of the project initialization logic
}

func runGraphicMode() {
	// Logic for running the GUI (to be implemented with a GUI library)
}

func init() {
	initCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Run the command in interactive mode")
	initCmd.Flags().BoolVarP(&graphic, "graphic", "g", false, "Run the command in graphic mode")
	
	initCmd.Flags().StringVar(&web, "web", "", "Specify a web target option (csr, ssr, ssg)")
	initCmd.Flags().StringVar(&mobile, "mobile", "", "Specify a mobile target option (android, ios)")
	initCmd.Flags().StringVar(&desktop, "desktop", "", "Specify a desktop target option (windows, darwin, linux)")
	initCmd.Flags().StringVar(&terminal, "terminal", "", "Specify a terminal target option (any additional terminal option can be added here)")
	
	rootCmd.AddCommand(initCmd)
}


var defaultprojectfile = `
package main

import (
	"github.com/atdiar/particleui"
	"github.com/atdiar/particleui/drivers/js"
	. "github.com/atdiar/particleui/drivers/js/declarative"
)

func App() doc.Document {

	document:= doc.NewDocument("HelloWorld", doc.EnableScrollRestoration()).EnableWasm()
	var input *ui.Element 
	var parahraph *ui.Element


	E(document.Body(),
		Children(
			E(document.Input.WithID("input").SetAttribute("type","text"),
				Ref(&input),
				// TODO need SetValueOnChange and SetValueOnInput vs SetValueOnEnter ?
			),
			E(document.Label().For(&input).SetText("What's your name?")),
			E(document.Paragraph().SetText("Hello!"),
				Ref(&parahraph),
		),
	)

	// The document observes the input for changes and update the paragraph accordingly.
	document.Watch("ui","text",input, ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		doc.ParagraphElement{paragraph}.SetText("Hello, "+evt.Value().String()+"!")
		return false
	}))
	return document
}

func main(){
	ListenAndServe := doc.NewBuilder(App)
	ListenAndServe(nil)
}

`
var defaultindexfile = `
<!doctype html>
<html>

<head>
	<meta charset="utf-8">
	<base href="/">
	
	<script id="wasmVM" src="/wasm_exec.js"></script>
	<script id="goruntime">
        let wasmLoadedResolver, loadEventResolver;
        window.wasmLoaded = new Promise(resolve => wasmLoadedResolver = resolve);
        window.loadEventFired = new Promise(resolve => loadEventResolver = resolve);

        window.onWasmDone = function() {
            wasmLoadedResolver();
        }

        window.addEventListener('load', () => {
            loadEventResolver();
        });

        const go = new Go();
        WebAssembly.instantiateStreaming(fetch("/main.wasm"), go.importObject)
        .then((result) => {
            go.run(result.instance);
        });

        Promise.all([window.wasmLoaded, window.loadEventFired]).then(() => {
            setTimeout(() => {
                window.dispatchEvent(new Event('PageReady'));
            }, 50);
        });
    </script>

</head>

<body>
	

</body>

</html>
`