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
var template string

var web, desktop, terminal bool
var mobile string 

var config map[string]string
const configFileName = "zui.config.json"

func configExists() bool {
	_, err := os.Stat(configFileName)
	return !os.IsNotExist(err)
}

//  Check that config is valid, i.e. it has at least the projectName and platform keys.
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
		only one target for that platform as seen in the mobile case.
	
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
			fmt.Println("Error: Project name is required.")
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
			fmt.Print(platformsSpecified, web,mobile,desktop,terminal)
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

		// git should ignore the release directory
		// TODO remove this?
		/*
		createFile(".gitignore", `
			/release
			/dev/build/*
			!/dev/build/app
		`)
		*/

		if web {
			// handle web project initialization
			config["projectName"] = projectName
			config["platform"] = "web"
			config["web"] = ""
			
			if template == ""{
				// project initialization logic
				// should create the directories, basic template file, dev directory with dev server runnable source

				// Create dev directory if it doesn't already exists
				err:= createDirectory(filepath.Join(".","dev"))
				if err!= nil{
					fmt.Println("Error: Unable to create dev directory.")
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

				
				
				err = createDirectory(filepath.Join(".","dev","build"))
				if err!= nil{
					fmt.Println("Error: unable to create dev/build directory.",err)
					os.Exit(1)
					return
				}

				err = createDirectory(filepath.Join(".","dev","build","app"))
				if err!= nil{
					fmt.Println("Error: Unable to create dev/build/app directory.")
					os.Exit(1)
					return
				}

				// Default main.go file
				err = createFile(filepath.Join(".","dev","main.go"), defaultprojectfile)
				if err!= nil{
					fmt.Println("Error: Unable to create dev/build/app/main.go file.")
					os.Exit(1)
					return
				}

				if verbose{
					fmt.Println("default main.go file created.")
				}

				// Default index.html file
				err = createFile(filepath.Join(".","dev","build","app","index.html"), defaultindexfile)
				if err!= nil{
					fmt.Println("Error: Unable to create dev/build/app/index.html file.")
					os.Exit(1)
					return
				}

				if verbose{
					fmt.Println("default index.html file created.")
				}

				// copy wasm_exec.js to the ./dev/build/app directory
				err = CopyWasmExecJs(filepath.Join(".","dev","build","app"))
				if err!= nil{
					fmt.Println("Error: Unable to copy wasm_exec.js file.",err)
					os.Exit(1)
					return
				}

				if verbose{
					fmt.Println("wasm_exec.js file copied from Go distribution.")
				}

				// Create asset folder and put a default favicon.ico in it
				err = createDirectory(filepath.Join(".","dev","build","app","assets"))
				if err!= nil{
					fmt.Println("Error: Unable to create dev/build/app/assets directory.")
					os.Exit(1)
					return
				}

				err = createFile(filepath.Join(".","dev","build","app","assets","favicon.ico"), "")
				if err!= nil{
					fmt.Println("Error: Unable to create dev/build/app/assets/favicon.ico file.")
					os.Exit(1)
					return
				}

				// Create directory for the HMR source code that can then be run
				// to watch over the app files and recompile


				// This should be a module, so run go mod init in the current directory.
				// The module name should be the project name.
				err = initGoModule(projectName)
				if err!= nil{
					fmt.Println("Error: Unable to initialize go module.",err)
					os.Exit(1)
					return
				}

				if verbose{
					fmt.Println("go module initialized.")
				}

				// Add the current directory to the workspace if GOWORK is set
				err = tryAddToWorkspace()
				if err!= nil{
					fmt.Println("Error: Unable to add to workspace.",err)
					os.Exit(1)
					return
				}
				
				if verbose{
					fmt.Println("added the project module to workspace.")
				}

			} else{
				// TODO
				// run $go new template_URL projectname
			}
			
			// Let's build the default app.
			// The output file should be in dev/build/app/main.wasm
			err := Build(filepath.Join(".","build","app", "main.wasm"),nil)
			if err != nil {
				fmt.Println("Error: Unable to build the default app.",err)
				os.Exit(1)
				return
			}

			if verbose{
				fmt.Println("default app built.")
			}

			// Let's build the default server.
			// The output file should be in dev/build/server/csr/
			err = Build(filepath.Join(".","build","server", "csr","main"),[]string{"server","csr"})
			if err != nil {
				fmt.Println("Error: Unable to build the default server.")
				os.Exit(1)
				return
			}

			if verbose{
				fmt.Println("default server built.")
			}

			// Config file should be valid now.
			if err := SaveConfig(); err != nil {
				fmt.Println("Error: Unable to save configuration file.")
				os.Exit(1)
				return
			}
			if verbose{
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
			} else{
				// TODO
			}
			fmt.Println("Mobile platform not yet implemented.")
			os.Exit(1)
		} else if desktop{
		
			// Process desktopOptions further
			if template != "" {
				// TODO
			} else{
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
			} else{
				// TODO
			}
			if err := SaveConfig(); err != nil {
				fmt.Println("Error: Unable to save configuration file.")
				os.Exit(1)
				return
			}
			if verbose{
				fmt.Println("SUCCESS! Your terminal project has been initialized.")
			}
		} else {
			fmt.Println("Error: A platform (web, mobile, desktop, terminal) must be specified.")
			os.Exit(1)
		}
	},	
}

func On(platform string) bool{
	_,ok:= config[platform]
	return ok
}

func createDirectory(path string) error{
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	if verbose {
		fmt.Printf("%s directory created.\n", path)
	}
	return nil
}

func createFile(path, content string) error {
    // Convert the content string to a byte slice
    data := []byte(content)

    // Write the data to the path, os.WriteFile handles creating or truncating the file
    err := os.WriteFile(path, data, 0644) // 0644 is a common permission setting for writable files
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

func initGoModule(moduleName string) error {
	// Check if the current directory is already a go module
	_, err := os.Stat("go.mod")
	if err == nil {
		if verbose{
			fmt.Println("go.mod already exists, skipping module initialization")
		}
		return err
	}
	cmd := exec.Command("go", "mod", "init", moduleName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error initializing go module: %s, output: %s", err, output)
	}
	if verbose{
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

func tryAddToWorkspace() error {
	ok,err := isGoWorkSet()
	if err != nil{
		return err
	}
    if ok {
        // Attempt to add the current directory to the workspace
        cmd := exec.Command("go", "work", "use", "-r", ".")
        output, err := cmd.CombinedOutput()
		if err != nil{
			return fmt.Errorf("error adding to workspace: %s, output: %s", err, output)
		}
    
		if verbose{
			fmt.Println("Successfully added to Go workspace")
		}

    } else {
		if verbose{
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

func Build(outputPath string, buildTags []string, cmdArgs ...string) error {
	if On("web"){
		// Check if the build is for WebAssembly and save the current environment
		isWasm := strings.HasSuffix(outputPath, ".wasm")

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
		//outputDir := filepath.Dir(outputPath)
		outputDirRel,err := filepath.Rel(filepath.Join(".","dev"),outputPath)
		if err!= nil{
			return err
		}
		outputDir := filepath.Dir(outputDirRel)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("error creating output directory: %v", err)
		}

		args := []string{"build"}

		// add ldflags if any relevant
		ldflags:= ldflags()
		if ldflags != "" {
			args = append(args, "-ldflags", ldflags)	
		}

		 // Add build tags if provided
		if len(buildTags) > 0 {
			args = append(args, "-tags", strings.Join(buildTags, " "))
		}
	
		// Add additional command-line arguments if provided
		if len(cmdArgs) > 0 {
			args = append(args, cmdArgs...)
		}
	
		// Set the output file
		args = append(args, "-o", outputPath)
	
		// Specify the source file
		sourceFile := "."
		args = append(args, sourceFile)

		if verbose{
			fmt.Println("Running go build",args)
		}

		// Execute the build command
		cmd := exec.Command("go", args...)
		cmd.Dir = filepath.Join(".","dev")

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if isWasm{
			cmd.Env = append(cmd.Environ(),"GOOS=js", "GOARCH=wasm")
		}

		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("build failed: %v", err)
		}
		
		return nil
	}

	if On("mobile"){
		// TODO
		// target aware (android vs ios)
		return fmt.Errorf("mobile platform not yet implemented")
	}

	if On("desktop"){
		// TODO
		return fmt.Errorf("desktop platform not yet implemented")
	}

	if On("terminal"){
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
	_,err:= fmt.Scanln(&input)
	if err != nil {
		fmt.Println("Error: Unable to read project name input.")
		os.Exit(1)
		return
	}
	projectName = input

	iloop:
	for{
		// Prompt for platform
		fmt.Print(`
		Choose a platform (1,2,3, or 4): 
			1. web
			2. mobile
			3. desktop
			4. terminal
			
		`)
		_,err = fmt.Scanln(&platform)
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
				_,err = fmt.Scanln(&target)
				if err != nil {
					fmt.Println("Error: Unable to read mobile target input.")
					os.Exit(1)
					return
				}
				switch target {
				case 1: mobile = "android"
					break platformloop
				case 2: mobile = "ios"
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
	
	initCmd.Flags().BoolVarP(&web, "web","w", false, "Specify a web target option (csr, ssr, ssg)")
	initCmd.Flags().StringVar(&mobile, "mobile", "", "Specify a mobile target option (android, ios)")
	initCmd.Flags().BoolVarP(&desktop, "desktop", "d", false, "Specify a desktop target option (windows, darwin, linux)")
	initCmd.Flags().BoolVarP(&terminal, "terminal", "t",false, "Specify a terminal target option (any additional terminal option can be added here)")
	initCmd.Flags().StringVar(&template, "template", "", "Specify a template URL to initialize the project from")
	
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
	var paragraph *ui.Element


	E(document.Body(),
		Children(
			E(document.Input.WithID("input", "text").SetAttribute("type","text"),
				Ref(&input),
				doc.SyncValueOnChange(),
			),
			E(document.Label().For(input.AsElement()).SetText("What's your name?")),
			E(document.Paragraph().SetText("Hello!"),
				Ref(&paragraph),
			),
		),
	)

	// The document observes the input for changes and update the paragraph accordingly.
	document.Watch("data","text",input, ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		doc.ParagraphElement{paragraph}.SetText("Hello, "+evt.NewValue().(ui.String).String()+"!")
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