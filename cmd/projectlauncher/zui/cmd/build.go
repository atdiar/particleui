/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var csr, ssr, ssg bool
var basepath string
var static string
var clientonly bool

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build is the command that builds the project",
	Long: `
		Build is the command that triggers a developpement build of the project.
		It does not prepare a new release build of the project.
		It simply creates an unoptimized executable for
		development purposes.

		Depending on the platform (web, mobile, desktop, terminal), the command may
		accept different flags.
		For example, the web platform accepts the following flags:
		- csr (default): compile the project for client-side rendering 
			o app is compiled as main.wasm and found in ./bin/tmp/client/{rootdirectory}/
			o the server is compiled as main (.exe on windows) and found in ./bin/tmp/server/csr/
			o the index page is rendered as index.html in ./bin/tmp/client/{rootdirectory}/

		- ssr: compile the project for server-side rendering
			o app is compiled as main.wasm and found in ./bin/tmp/client/{rootdirectory}/
			o the server is compiled as main (.exe on windows) and found in ./bin/tmp/server/ssr/

		- ssg: compile the project, producing the static html files (static site generation)
			o the different pages are found in ./bin/tmp/client/{rootdirectory}/
			o the server is compiled as main (.exe on windows) and found in ./bin/tmp/server/ssg/

		- static: compile a specific page of the project when in csr or ssg mode.
			In order to output the html file, the server build is invoked.

		- "client": the command will only build the client.
			It is needed since the web platform builds both the client and the server by default.

		The mobile platform has its build target specified at initialization time.
		It does not need to be supplied at build time.

		The desktop and terminal platforms have their build target determined by the OS the command is run on.
		Nothing additional needs to be supplied.
	`,
	Example: `
		# building a web project
		zui build -csr
		zui build -ssr
		zui build -ssg

		# building a page for a web project
		zui build -csr -static= '/' --> renders the index page as index.html in /_root/ or /{basepath}/ if a basepath command line argument is also passed.
		zui build -csr -static= '/people/list/partners' ==> renders the page as index.html in /bin/tmp/client/_root/people/list/partners/ or /bin/tmp/client/{basepath}/people/list/partners/ if a basepath is applicable as previously.

		# building a project with a given basepath
		zui build -basepath=/path
		The path needs to use a leading slash so as to be relative to the root.


		# building a mobile, desktop or terminal  project
		zui build

		
		
	`,
	Run: buildFunc,
}

var buildFunc = func(cmd *cobra.Command, args []string) {
	err := LoadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}

	if On("web") {
		// basepath needs to be validated
		// It needs to start with a slash
		b, err := url.Parse(basepath)
		if err != nil {
			fmt.Println("invalid basepath")
			os.Exit(1)
			return
		}
		if b.Path != "/" {
			if b.Path[0] != '/' {
				fmt.Println("invalid basepath: basepath needs to start with a slash")
				os.Exit(1)
				return
			}
		}

		// TODO implement zui build -csr -clean, that will erase the content
		// of the bin/tmp/client/{rootdirectory} and bin/tmp/server/csr/{rootdirectory} directories
		// before (re)building.

		// if csr
		if csr {
			err = Build(true, nil)
			if err != nil {
				fmt.Println("Error: unable to build the app.", err)
				os.Exit(1)
				return
			}

			if verbose {
				fmt.Println("default app built.")
			}

			if !clientonly {
				// Let's build the default server.
				// The output file should be in /bin/tmp/server/csr/
				err = Build(false, []string{"server", "csr"})
				if err != nil {
					fmt.Println("Error: unable to build the default server.")
					os.Exit(1)
					return
				}
				if verbose {
					fmt.Println("csr server built.")
				}
			}

			// TODO using the special we should be able to generate the index page, taking into account basepath etc.
			// The output directory for the rendered html file is /bin/tmp/client/{rootdirectory}
			// The output directory for the rendering server is /bin/tmp/server
			// This is for csr mode only.

			rootdirectory := "_root"
			if basepath != "/" {
				rootdirectory = basepath // remove the leading slash? DEBUG
			}

			// We can copy the assets from the source directory to the output directory.
			outputDir := filepath.Join(".", "bin", "tmp", "client", rootdirectory)
			err = copyDirectory(filepath.Join(".", "src", "assets"), filepath.Join(outputDir, "assets"))
			if err != nil {
				fmt.Println("Error: unable to copy assets to the output directory.")
				os.Exit(1)
				return
			}
			if verbose {
				fmt.Println("assets copied to the output directory.")
			}

			err = renderPages("/", releaseMode || tinygo)
			if err != nil {
				fmt.Println("Error: unable to render the index page.", err)
				os.Exit(1)
				return
			}

			// TODO -static flag handling
			// everything that is rendered as a file is served statically with higher priority. (the server needs to check on startup and implement the shortcircuit logic)
			if static != "" {
				err = renderPages(static, releaseMode || tinygo)
				if err != nil {
					fmt.Println("Error: unable to render the page at", static, err)
					os.Exit(1)
					return
				}
			}

			if verbose {
				fmt.Println("Build successful.")
			}

		} else if ssr {
			err = Build(true, nil)
			if err != nil {
				fmt.Println("Error: unable to build the app.", err)
				os.Exit(1)
				return
			}

			if verbose {
				fmt.Println("wasm app built.")
			}

			if clientonly {
				err = Build(false, []string{"server", "ssr"})
				if err != nil {
					fmt.Println("Error: unable to build the ssr server.")
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("ssr server built.")
				}
			}

			rootdirectory := "_root"
			if basepath != "/" {
				rootdirectory = basepath // remove the leading slash? DEBUG
			}

			// We can copy the assets from the source directory to the output directory.
			outputDir := filepath.Join(".", "bin", "tmp", "client", rootdirectory)
			err = copyDirectory(filepath.Join(".", "src", "assets"), filepath.Join(outputDir, "assets"))
			if err != nil {
				fmt.Println("Error: unable to copy assets to the output directory.")
				os.Exit(1)
				return
			}
			if verbose {
				fmt.Println("assets copied to the output directory.")
			}

			// TODO -static flag handling
			// everything that is rendered as a file is served statically with higher priority. (the server needs to check on startup and implement the shortcircuit logic)
			if static != "" {
				err = renderPages(static, releaseMode || tinygo)
				if err != nil {
					fmt.Println("Error: unable to render the page at", static, err)
					os.Exit(1)
					return
				}
			}

			if verbose {
				fmt.Println("Build successful.")
			}

		} else if ssg {
			err = Build(false, []string{"server", "ssg"})
			if err != nil {
				fmt.Println("Error: unable to build the ssg server.")
				os.Exit(1)
				return
			}

			if verbose {
				fmt.Println("ssg server built.")
			}

			rootdirectory := "_root"
			if basepath != "/" {
				rootdirectory = basepath // DEBUG remove the leading slash if any?
			}

			// We can copy the assets from the source directory to the output directory.
			outputDir := filepath.Join(".", "bin", "tmp", "client", rootdirectory)
			err = copyDirectory(filepath.Join(".", "src", "assets"), filepath.Join(outputDir, "assets"))
			if err != nil {
				fmt.Println("Error: unable to copy assets to the output directory.")
				os.Exit(1)
				return
			}
			if verbose {
				fmt.Println("assets copied to the output directory.")
			}

			// TODO -static flag handling
			// if empty, renders every page
			// otherwise, renders the specified page(s)
			err = renderPages(static, releaseMode || tinygo)
			if err != nil {
				fmt.Println("Error: unable to render the page at", static, err)
				os.Exit(1)
				return
			}

			if verbose {
				fmt.Println("Build successful.")
			}

		}
	} else if On("mobile") {
		// TODO
		// Make sure that only acceptable flags have been passed.
		// csr, ssr, ssg don't make any sense here.
		fmt.Println("building for mobile is not yet supported")
		os.Exit(1)
	} else if On("desktop") {
		// TODO
		// Make sure that only acceptable flags have been passed.
		// csr, ssr, ssg don't make any sense here.
		fmt.Println("building for desktop is not yet supported")
		os.Exit(1)
	} else if On("terminal") {
		// TODO
	} else {
		fmt.Println("unknown platform")
		os.Exit(1)
		return
	}
}

func renderPages(renderPath string, releasebuild bool) error {
	if verbose {
		fmt.Println("rendering pages...")
		fmt.Println("renderPath: ", renderPath)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current working directory: %w", err)
	}

	rootdirectory := "_root"
	if basepath != "/" {
		if verbose {
			fmt.Println("basepath is: ", basepath)
		}
		rootdirectory = basepath
	}

	pathToServerBinary := getServerBinaryPath("csr", releasebuild, rootdirectory)
	if ssr {
		pathToServerBinary = getServerBinaryPath("ssr", releasebuild, rootdirectory)
	}
	if ssg {
		pathToServerBinary = getServerBinaryPath("ssg", releasebuild, rootdirectory)
	}

	// Convert to absolute path
	absoluteBinaryPath := filepath.Join(cwd, pathToServerBinary)

	// Check if the binary exists using absolute path
	if _, err := os.Stat(absoluteBinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("server binary does not exist at path: %s", absoluteBinaryPath)
	}

	fmt.Println("about to render.......")
	// Use the absolute path
	cmd := exec.Command(absoluteBinaryPath, "--render", renderPath)
	if basepath != "" {
		cmd.Args = append(cmd.Args, "--basepath", basepath)
	}

	if verbose {
		cmd.Args = append(cmd.Args, "--verbose")
	}

	// Set working directory to where the binary lives
	cmd.Dir = filepath.Dir(absoluteBinaryPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("OUTPUT ==============")
		fmt.Println(string(output))
		fmt.Println("OUTPUT ==============")
		return fmt.Errorf("render failed: %s", err)
	}

	if verbose {
		fmt.Println("output: ", string(output), renderPath)
	}

	if verbose {
		fmt.Println("SUCCESS: rendered page ", renderPath)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&basepath, "basepath", "", "/", "base path for the project")
	buildCmd.Flags().BoolVarP(&csr, "csr", "c", false, "build for client-side rendering")
	buildCmd.Flags().BoolVarP(&ssr, "ssr", "s", false, "build for server-side rendering")
	buildCmd.Flags().BoolVarP(&ssg, "ssg", "g", false, "build for static site generation")
	buildCmd.Flags().StringVarP(&static, "static", "", "", "build one or several pages of the project. If none are explicitly specified, using this flag builds the root.")
	buildCmd.Flags().BoolVarP(&clientonly, "client", "", false, "build only the client (default is to build both client and server)")
	buildCmd.Flags().BoolVarP(&releaseMode, "release", "r", false, "build in release mode")
	buildCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	buildCmd.Flags().BoolVarP(&nolr, "nolr", "", false, "disable live reloading")

	if !csr && !ssr && !ssg {
		// If none of the flags are set, we default to csr
		csr = true
	}
}
