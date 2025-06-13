/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var verbose bool

var interactive, graphic bool
var projectName string
var template string

var web, desktop, terminal bool
var mobile string

var config map[string]string

const configFileName = "zui.config.json"

func configExists() bool {
	_, err := os.Stat(configFileName)
	return !os.IsNotExist(err)
}

// Check that config is valid, i.e. it has at least the projectName and platform keys.
func configIsValid() bool {
	if _, ok := config["projectName"]; !ok {
		return false
	}
	if _, ok := config["platform"]; !ok {
		return false
	}
	return true
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

	if !configIsValid() {
		return fmt.Errorf("invalid configuration file")
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

func createNewProject() error {
	dirs := []string{
		"src",
		"src/assets",
		"src/assets/images",
		"src/assets/styles",
		"src/assets/scripts",
		"bin",
	}

	for _, dir := range dirs {
		if err := createDirectory(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	gitignoreContent := `# Build directories
/bin/tmp/

# OS specific files
.DS_Store
Thumbs.db
*.swp
*.swo

# IDE specific
.idea/
.vscode/
*.sublime-*

# Go specific
*.exe
*.test
*.prof`

	if err := createFile(".gitignore", gitignoreContent); err != nil {
		return fmt.Errorf("failed to create .gitignore: %v", err)
	}

	return nil
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init command is used to launch a new GUI project",
	Long: `init is the initialization command for a new GUI project.
		It creates the project structure and the configuration files.
		The project should be named, typically by providing the
		URL of the project repository.
		It accepts the platform as mandatory argument, among which are:
		- web
		- mobile
		- desktop
		- terminal
		
		Some platforms allow different build options, such as web:
			o csr (client-side rendering)
			o ssr (server-side rendering)
			o ssg (static site generation)
		

		Some other platforms require the build target to be specified at initialization time:
		Such is the case for the mobile platform:
			o android
			o ios
		In fact, the whole project depends on the platform and the target in this case.

		Lastly, some projects are fully platform-agnostic, such as desktop or terminal projects.
		Depending on the OS the commands are run on, they will allow to build the corresponding binary 
		for either of the following OSes:
			o windows
			o linux
			o macOS (darwin)
		
		An initialized project may only target one platform, and sometimes even
		only one target for that platform as seen in the mobile case where it's either iOS
		or android but not both.
	
	`,
	Example: `
		zui init github.com/stephenstrange/thewebapp --web
		zui init github.com/stephenstrange/theiosapp --mobile=ios
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
			runInteractiveMode() // if call to zui init without arguments, run the interactive mode
			return
		}

		platformsSpecified := 0

		if web {
			platformsSpecified++
		}
		if mobile != "" {
			platformsSpecified++
		}
		if desktop {
			platformsSpecified++
		}
		if terminal {
			platformsSpecified++
		}

		if platformsSpecified > 1 {
			fmt.Println("Error: Please specify only one platform (web, mobile, desktop, terminal).")
			fmt.Print(platformsSpecified, web, mobile, desktop, terminal)
			os.Exit(1)
			return
		}

		// TODO: Check that the project config file is valid i.e. the initialization has been correctly done.

		if configExists() {
			fmt.Println("Error: A project already exists in this directory.")
			os.Exit(1)
			return
		}

		config = make(map[string]string)

		if web {
			// handle web project initialization
			config["projectName"] = projectName
			config["platform"] = "web"
			config["web"] = ""

			if template == "default" {
				// project initialization logic
				// should create the directories, default template file, dev directory with dev server runnable source

				// Check that the project directory is empty.
				// If the project directory is not empty,
				// the only files that should be found are the go.mod file and if then,  go.sum
				files, err := os.ReadDir(".")
				if err != nil {
					fmt.Println("Error: Unable to read project directory.")
					os.Exit(1)
					return
				}
				if len(files) > 0 {
					var hasgomod bool
					for _, file := range files {
						if file.Name() == "go.mod" {
							hasgomod = true
							continue
						} else if file.Name() == "go.sum" {
							continue
						} else if file.Name() == ".gitignore" {
							continue
						} else if file.Name() == ".git" {
							continue
						} else {
							fmt.Println("Error: Project directory is not empty.", file)
							os.Exit(1)
							return
						}
					}
					if !hasgomod {
						fmt.Println("Error: Project directory is not empty.")
						os.Exit(1)
						return
					}
				}

				// Create dev directory if it doesn't already exists
				err = createNewProject()
				if err != nil {
					fmt.Println("Error: Unable to create new project.\n", err)
					os.Exit(1)
					return
				}

				// /src holds the source code for the app.
				//
				// zui build compiles in CSR mode by default.
				// It also builds the server executable in /bin/tmp/server/csr/
				//
				// zui build --ssr compiles the server executable in /bin/tmp/server/ssr/
				// While a client-side rendering server binary defers HTML rendering to the client,
				// an server-side rendering server binary generates the HTML page on the first hit.
				//
				//
				// zui build --ssg "." compiles the full site, rendering static files in /bin/tmp/server/ssg.
				// zui build --ssg "/" builds the index page only etc.
				// To note that the entry point iindex.html file will be located in the /_root/ directory or if
				// a basepath is specified, in the /basepath/ directory.
				//
				// zui run -dev starts the dev server (default is csr mode).
				// zui run -dev -ssr starts the dev server in SSR mode.
				// zui run -dev -ssg starts the dev server in SSG mode.
				// It serves the files in /bin/tmp/client/_root/ directory (respectively /bin/tmp/client/{basepath} if applicable).
				//
				// -port allows to change the port number for the development server.
				//

				// Default build: on project initialization, a default project is
				// created and built in CSR mode. (unless --template=none is specified)
				// A default app is built from a template and can be run by the command zui run -dev.
				//
				// In the future, it should be possible to run zui init --web -template= template_URL
				// to create a project from a template. (TODO: use go new)

				// Let's create the default main.go file in the /src directory.
				// This will contain a default app that outputs a hello world, a game or something.
				// The default app should be a module, so run go mod init in the current directory.
				// The module name should be the project name.

				// Default main.go file
				err = createFile(filepath.Join(".", "src", "main.go"), defaultprojectfile)
				if err != nil {
					fmt.Println("Error: Unable to create src/main.go file.")
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("default main.go file created.")
				}

				// TODO replace the favicon with the default one for zui
				err = createFile(filepath.Join(".", "src", "assets", "favicon.ico"), "")
				if err != nil {
					fmt.Println("Error: Unable to create src/assets/favicon.ico file.")
					os.Exit(1)
					return
				}

				// This should be a module, so run go mod init in the current directory.
				// The module name should be the project name.
				err = initGoModule(projectName)
				if err != nil {
					fmt.Println("Error: Unable to initialize go module.", err)
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("go module initialized.")
				}

				// Add the current directory to the workspace if GOWORK is set
				err = tryAddToWorkspace()
				if err != nil {
					fmt.Println("Error: Unable to add to workspace.", err)
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("added the project module to workspace.")
				}

			} else {
				// TODO
				// run $go new template_URL projectname
			}

			// TODO build the default project in dev mode with HMR enabled ****************
			// the index.html needs to be generated.

			// Config file should be valid now.
			if err := SaveConfig(); err != nil {
				fmt.Println("Error: Unable to save configuration file.")
				os.Exit(1)
				return
			}
			if verbose {
				fmt.Println("SUCCESS! Your web project has been initialized.")
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
			// Process mobileOptions further TODO
			if template != "" {
				// TODO
			} else {
				// TODO
			}
			fmt.Println("Mobile platform not yet implemented.")
			os.Exit(1)
		} else if desktop {

			// Process desktopOptions further
			if template != "" {
				// TODO
			} else {
				// TODO
			}

			fmt.Println("Desktop platform not yet implemented.")
			os.Exit(1)
		} else if terminal {
			// handle terminal initialization
			// TODO initialize default terminal example app
			config["projectName"] = projectName
			config["platform"] = "terminal"
			config["terminal"] = ""
			if template != "" {
				// TODO
			} else {
				// TODO
			}
			if err := SaveConfig(); err != nil {
				fmt.Println("Error: Unable to save configuration file.")
				os.Exit(1)
				return
			}
			if verbose {
				fmt.Println("SUCCESS! Your terminal project has been initialized.")
			}
		} else {
			fmt.Println("Error: A platform (web, mobile, desktop, terminal) must be specified.")
			os.Exit(1)
		}
	},
}

func On(platform string) bool {
	_, ok := config[platform]
	return ok
}

func createDirectory(path string) error {
	path, err := filepath.Rel(filepath.Join("."), path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	if verbose {
		fmt.Printf("%s directory created.\n", path)
	}
	return nil
}

func createFile(path, content string) error {
	path, err := filepath.Rel(filepath.Join("."), path)
	if err != nil {
		return err
	}

	// Convert the content string to a byte slice
	data := []byte(content)

	// Write the data to the path, os.WriteFile handles creating or truncating the file
	err = os.WriteFile(path, data, 0644) // 0644 is a common permission setting for writable files
	if err != nil {
		return err
	}

	// Verbose output
	if verbose {
		fmt.Printf("%s file created or overwritten.\n", path)
	}
	return nil
}

// CopyWasmExecJs copies the wasm_exec.js file from the Go distribution to the specified destination directory.
func CopyWasmExecJs(destinationDir string) error {
	// Determine the Go root directory
	goRoot := runtime.GOROOT()

	// Source wasm_exec.js path
	source := filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js")

	// Ensure the destination directory exists
	err := os.MkdirAll(destinationDir, 0755) // Create the destination directory if it does not exist
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

func CopyWasmExecJsTinygo(destinationDir string) error {
	// get the file from where it should be installed i.e. /usr/local/lib/tinygo/targets/wasm_exec.js
	// Source wasm_exec.js path
	source := filepath.Join("/usr/local/lib/tinygo/targets", "wasm_exec.js")
	err := os.MkdirAll(destinationDir, 0755) // Create the destination directory if it does not exist
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
	if verbose {
		fmt.Println("wasm_exec.js file copied from tinygo distribution.")
	}
	return nil
}

func initGoModule(moduleName string) error {
	// Check if the current directory is already a go module
	_, err := os.Stat("go.mod")
	if err == nil {
		if verbose {
			fmt.Println("go.mod already exists, skipping module initialization")
		}
		return err
	}
	cmd := exec.Command("go", "mod", "init", moduleName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error initializing go module: %s, output: %s", err, output)
	}
	if verbose {
		fmt.Printf("Successfully initialized go module: %s\n", moduleName)
	}

	return nil
}

// copyFile is a helper function that copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Get source file info for permissions
	stat, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Sync to ensure write to disk
	err = destFile.Sync()
	if err != nil {
		return err
	}

	// Set the same permissions as source file
	return os.Chmod(dst, stat.Mode())
}

func tryAddToWorkspace() error {
	ok, err := isGoWorkSet()
	if err != nil {
		return err
	}
	if ok {
		// Attempt to add the current directory to the workspace
		cmd := exec.Command("go", "work", "use", "-r", ".")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error adding to workspace: %s, output: %s", err, output)
		}

		if verbose {
			fmt.Println("Successfully added to Go workspace")
		}

	} else {
		if verbose {
			fmt.Println("GOWORK is not set, skipping workspace addition")
		}
	}
	return nil
}

func isGoWorkSet() (bool, error) {
	out, err := exec.Command("go", "env", "GOWORK").Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func getServerBinaryPath(serverType string, releaseBuild bool, rootDirectory string) string {
	base := "tmp"
	if releaseBuild {
		base = "release"
	}

	path := filepath.Join(".", "bin", base, "server", serverType, rootDirectory, "main")

	// Handle Windows executable extension
	if runtime.GOOS == "windows" {
		path += ".exe"
	}

	return path
}

func Build(client bool, buildTags []string, cmdArgs ...string) error {
	if On("web") {

		toolchain := "go"
		// input path is the current directory which should correspond to /src from the project root
		// i.e. filepath.Join(".", "src")
		// It holds the source code.
		//
		// The output path is also known since the project structure remains fixed:
		// Building the client in development mode will output the files in /bin/tmp/client/_root/ or /bin/tmp/client/{basepath} if a basepath is specified.
		// Building the server in development mode will output the files in /bin/tmp/server/csr/ or /bin/tmp/server/ssr/ or /bin/tmp/server/ssg/ depending on the server type
		// mentioned in the buildtags.
		// The server binaries can be executed in a basepath-aware mode via linker flags ldflags, handled by the custom build command options.
		// In release mode, replace 'tmp' by 'release'.

		rootdirectory := "_root"
		if basepath != "/" {
			if verbose {
				fmt.Println("basepath is: ", basepath)
			}
			rootdirectory = filepath.Join(rootdirectory, basepath)
		}

		var outputPath string
		if client {
			if releaseMode {
				outputPath = filepath.Join(".", "bin", "release", "client", rootdirectory, "main.wasm")
			} else {
				outputPath = filepath.Join(".", "bin", "tmp", "client", rootdirectory, "main.wasm")
			}
		} else {
			if csr {
				outputPath = getServerBinaryPath("csr", releaseMode, rootdirectory)
			} else if ssr {
				outputPath = getServerBinaryPath("ssr", releaseMode, rootdirectory)
			} else if ssg {
				outputPath = getServerBinaryPath("ssg", releaseMode, rootdirectory)
			}
		}

		// Determine the correct file extension for the executable for non-WASM builds
		if !client {
			goos := os.Getenv("GOOS")
			if goos == "" {
				goos = runtime.GOOS // Default to the current system's OS if GOOS is not set
			}
			if goos == "windows" && !strings.HasSuffix(outputPath, ".exe") {
				outputPath += ".exe"
			}
		} else {
			if tinygo {
				// let's check whether the tinygo toolchain is available, otherwise error out
				_, err := exec.LookPath("tinygo")
				if err != nil {
					return fmt.Errorf("tinygo is not installed")
				}
				toolchain = "tinygo"
			}
		}
		outputDir := filepath.Dir(outputPath)
		if verbose {
			fmt.Println("output directory is: ", outputDir)
		}

		if tinygo {
			err := CopyWasmExecJsTinygo(outputDir)
			if err != nil {
				return fmt.Errorf("failed to copy wasm_exec.js: %v", err)
			}
		} else {
			if client {
				err := CopyWasmExecJs(outputDir)
				if err != nil {
					return fmt.Errorf("failed to copy wasm_exec.js: %v", err)
				}
			}
		}

		// TODO
		if client && ssg {
			// generate the pages that will be found in /bin/tmp/client/_root/ or /bin/tmp/client/{basepath}/
			// It consists in generating the page from a specific build of the ssg server hidden behind
			// a build tag which will render the pages to files in the output directory.
			// the ssg server itself is different as it is simply a fileserving server that is
			// basepath aware.
		}

		args := []string{"build"}

		// add ldflags if any relevant
		if ldflags := ldflags(); ldflags != "" {
			if releaseMode {
				if !tinygo {
					ldflags = "-s -w " + ldflags
				}
			}
			args = append(args, "-ldflags="+ldflags)
		}

		// Add build tags if provided
		if len(buildTags) > 0 {
			args = append(args, "-tags", strings.Join(buildTags, " "))
		}

		// Add additional command-line arguments if provided
		if len(cmdArgs) > 0 {
			args = append(args, cmdArgs...)
		}

		// Set the path to the output file
		args = append(args, "-o", filepath.Join("..", outputPath))

		// Specify the source file
		sourceFile := "."
		args = append(args, sourceFile)

		if verbose {
			fmt.Println("Running go build", args)
		}

		// Execute the build command
		cmd := exec.Command(toolchain, args...)
		cmd.Dir = filepath.Join(".", "src")

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if client {
			cmd.Env = append(cmd.Environ(), "GOOS=js", "GOARCH=wasm")
		}

		if tinygo {
			cmdArgs = append(cmdArgs, "no-debug", "=target=wasm", "-gc=conservative")
		}

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("build failed: %v", err)
		}

		return nil
	}

	if On("mobile") {
		// TODO
		// target aware (android vs ios)
		return fmt.Errorf("mobile platform not yet implemented")
	}

	if On("desktop") {
		// TODO
		return fmt.Errorf("desktop platform not yet implemented")
	}

	if On("terminal") {
		// TODO
		return fmt.Errorf("terminal platform not yet implemented")
	}

	return fmt.Errorf("unknown platform")
}

func runInteractiveMode() {
	var input string
	var platform int
	var target int

	// Prompt for project name
	fmt.Print("Project name: ")
	_, err := fmt.Scanln(&input)
	if err != nil {
		fmt.Println("Error: Unable to read project name input.")
		os.Exit(1)
		return
	}
	projectName = input

iloop:
	for {
		// Prompt for platform
		fmt.Print(`
		Choose a platform (1,2,3, or 4): 
			1. web
			2. mobile
			3. desktop
			4. terminal
			
		`)
		_, err = fmt.Scanln(&platform)
		if err != nil {
			fmt.Println("Error: Unable to read platform input.")
			os.Exit(1)
			return
		}

		switch platform {
		case 1:
			web = true
			break iloop
		case 2:
		platformloop:
			for {
				fmt.Print("Choose a target for mobile (1 for android, 2 for iOS): ")
				_, err = fmt.Scanln(&target)
				if err != nil {
					fmt.Println("Error: Unable to read mobile target input.")
					os.Exit(1)
					return
				}
				switch target {
				case 1:
					mobile = "android"
					break platformloop
				case 2:
					mobile = "ios"
					break platformloop
				default:
					fmt.Println("Invalid mobile target selected. Try again.")
				}
			}
			break iloop
		case 3:
			desktop = true
			break iloop
		case 4:
			terminal = true
			break iloop
		default:
			fmt.Println("Invalid platform selected. Try again.")
		}
	}

	// Continue with the rest of the project initialization logic
}

func runGraphicMode() {
	// Logic for running the GUI (to be implemented with a GUI library)
}

func init() {
	initCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Run the command in interactive mode")
	initCmd.Flags().BoolVarP(&graphic, "graphic", "g", false, "Run the command in graphic mode")

	initCmd.Flags().BoolVarP(&web, "web", "w", false, "Specify a web target option (csr, ssr, ssg)")
	initCmd.Flags().StringVar(&mobile, "mobile", "", "Specify a mobile target option (android, ios)")
	initCmd.Flags().BoolVarP(&desktop, "desktop", "d", false, "Specify a desktop target option (windows, darwin, linux)")
	initCmd.Flags().BoolVarP(&terminal, "terminal", "t", false, "Specify a terminal target option (any additional terminal option can be added here)")
	initCmd.Flags().StringVar(&template, "template", "default", "Specify a template URL to initialize the project from")

	rootCmd.AddCommand(initCmd)
}

var defaultprojectfile = `
package main

import (
	"github.com/atdiar/particleui"
	. "github.com/atdiar/particleui/drivers/js"
	"time"
)

func App() *Document {

	document:= NewDocument("HelloWorld", EnableScrollRestoration())
	
	var input *ui.Element 
	var paragraph *ui.Element
	var clock *ui.Element

	
	// the clock element is a ParagraphElement updating every second and which displays the current date.
	var DisplayDate = ui.OnMutation(func(evt ui.MutationEvent)bool{
		// Displays the local date and time
		ParagraphElement{evt.Origin()}.SetText("Today is: "+time.Now().Format("2006-01-02 15:04:05")+"\n")
		return false
	})
	

	E(document.Body(),
		Children( 
			E(document.Paragraph(),
				Ref(&clock),
				Modifier.OnTick(1* time.Second, DisplayDate),
			),
			E(document.Label().For(&input).SetText("What's your name?\n")),
			E(document.Input.WithID("input", "text"),
				Ref(&input),
				SyncValueOnInput(WithStrConv),
			),
			E(document.Paragraph().SetText("Hello!"),
				Ref(&paragraph),
			),
		),
	)

	// The document observes the input for changes and update the paragraph accordingly.
	document.Watch("data","value",input, ui.OnMutation(func(evt ui.MutationEvent)bool{
		ParagraphElement{paragraph}.SetText("Hello, "+ evt.NewValue().(ui.String).String() + "!")
		return false
	}))
	
	return document
}

func main(){
	ListenAndServe := NewBuilder(App)
	ListenAndServe(nil)
}
`

// TODO this should be part of the go code and rendered as html at build time in csr mode.
// At that point, this variable will be removed. We will only need go files with the html renderer being used
// at build time.
var defaultindexfile = `
<!doctype html>
<html>

<head>
	<meta charset="utf-8">
	<base id="zuibase" href=` + basepath + `>
	
	<script id="wasmVM" src="./wasm_exec.js"></script>
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
        WebAssembly.instantiateStreaming(fetch("./main.wasm"), go.importObject)
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
