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

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build is the command to build the project",
	Long: `
		Build is the command to build the project.
		It does not prepare a new release of the project.
		It simply builds the source and create an unoptimized executable for
		development purposes.

		Depending on the platform (web, mobile, desktop, terminal), the command may
		accept different flags.
		For example, the web platform accepts the following flags:
		- csr (default): compile the project for client-side rendering 
			o app is compiled as main.wasm and found in ./dev/build/app/
			o the server is compiled as main (.exe on windows) and found in ./dev/build/server/csr

		- ssr: compile the project for server-side rendering
			o app is compiled as main.wasm and found in ./dev/build/app/ (same as in csr)
			o the server is compiled as main (.exe on windows) and found in ./dev/build/server/ssr

		- ssg: compile the project, producing the static html files (static site generation)
			o the different pages are found in ./dev/build/ssg/pages/
			o the server is compiled as main (.exe on windows) and found in ./dev/build/server/ssg

		The mobile platform has its build target specified at initialization time.
		It does not need to be supplied at build time.

		The desktop and terminal platforms have their build target determined by the OS the command is run on.
	`,
	Example: `
		# building a web project
		zui build -csr
		zui build -ssr
		zui build -ssg

		# building a mobile, desktop or terminal  project
		zui build

		# building a project with a given basepath
		zui build -basepath=/path
		The path needs to use a leading slash so as to be relative to the root.

		
	`,
	Run: func(cmd *cobra.Command, args []string) {
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
			if !csr && !ssr && !ssg {
				// Prompt the user to choose an option
				fmt.Printf("Please choose one of the following options:\n")
				fmt.Printf("  1 for CSR (client-side rendering)\n  2 for SSR (server-side rendering)\n 3 for SSG (static site generation)\n")

				// Read the user's input
				var option int
				for {
					fmt.Scanf("%d", &option)

					// If the user enters a valid option, set the value of the CSR or SSR flag and break out of the loop
					if option == 1 {
						csr = true
						break
					}
					if option == 2 {
						ssr = true
						break
					}

					if option == 3 {
						ssg = true
						break
					}

					// Otherwise, display feedback and loop back to the prompt again
					fmt.Fprintf(os.Stderr, "Invalid option: %d\n", option)
				}
			}

			// if csr
			if csr {
				err = Build(true, nil)
				if err != nil {
					fmt.Println("Error: unable to build the default app.")
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("default app built.")
				}

				var buildall bool
				for _, a := range args {
					if a == "." {
						buildall = true
					}
				}

				if buildall {
					// Let's build the default server.
					// The output file should be in dist/server/csr/tmp/ or dist/server/csr/release/
					err = Build(false, []string{"server", "csr"})
					if err != nil {
						fmt.Println("Error: unable to build the default server.")
						os.Exit(1)
						return
					}
					if verbose {
						fmt.Println("default server built.")
					}
				}

				// TODO using the server build, we should be able to generate the index page, taking into account basepath etc.
				// But only if it hasn't been generated yet.
				// The output directory is dist/client/tmp/ or dist/client/release/

				// 1. let's find out if the index.html file exists
				folder := ".root"
				if basepath != "/" {
					folder = basepath[1:]
				}

				var indexhtmlExists bool
				if releaseMode {
					_, err = os.Stat(filepath.Join(".", "dist", "client", folder, "release", "index.html"))
					if err == nil {
						indexhtmlExists = true
					}
				} else {
					_, err = os.Stat(filepath.Join(".", "dist", "client", folder, "tmp", "index.html"))
					if err == nil {
						indexhtmlExists = true
					}
				}
				if indexhtmlExists {
					if verbose {
						fmt.Println("index.html exists. No need to render it.")
						fmt.Println("To force a re-render, delete the index.html file.")
						fmt.Println("Build successful.")
					}
					return
				}

				// 2. The index.html file does not exist, we need to render it.
				outputDir := filepath.Join(".", "dist", "client", folder, "tmp")
				if releaseMode {
					outputDir = filepath.Join(".", "dist", "client", folder, "release")
				}

				renderPages("/", outputDir, releaseMode)

			} else if ssr {
				err = Build(true, nil)
				if err != nil {
					fmt.Println("Error: unable to build the default app.")
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("wasm app built.")
				}

				// Let's build the default server.
				// The output file should be in in dist/server/ssr/tmp/ or dist/server/ssr/release/
				err = Build(false, []string{"server", "ssr"})
				if err != nil {
					fmt.Println("Error: unable to build the ssr server.")
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("ssr server built.")
				}
				// TODO using the server build, we should be able to generate the index page, taking into acocunt basepath etc.
				folder := ".root"
				if basepath != "/" {
					folder = basepath[1:]
				}

				var indexhtmlExists bool
				if releaseMode {
					_, err = os.Stat(filepath.Join(".", "dist", "client", folder, "release", "index.html"))
					if err == nil {
						indexhtmlExists = true
					}
				} else {
					_, err = os.Stat(filepath.Join(".", "dist", "client", folder, "tmp", "index.html"))
					if err == nil {
						indexhtmlExists = true
					}
				}
				if indexhtmlExists {
					if verbose {
						fmt.Println("index.html exists. No need to render it.")
						fmt.Println("To force a re-render, delete the index.html file.")
						fmt.Println("Build successful.")
					}
					return
				}
				// 2. The index.html file does not exist, we need to render it.
				outputDir := filepath.Join(".", "dist", "client", folder, "tmp")
				if releaseMode {
					outputDir = filepath.Join(".", "dist", "client", folder, "release")
				}

				renderPages("/", outputDir, releaseMode)

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

				// Now we need to build the pages by running the server executable
				// at least once.
				// The output files will be found in ./dist/client/.ssg/
				pathtoserverbin := filepath.Join(".", "dist", "server", "ssg", "tmp", "main")
				if releaseMode {
					pathtoserverbin = filepath.Join(".", "dist", "server", "ssg", "release", "main")
				}
				outputDir := filepath.Join(".", "dist", "client", ".ssg", "tmp")
				if releaseMode {
					outputDir = filepath.Join(".", "dist", "client", ".ssg", "release")
				}
				command := exec.Command(pathtoserverbin, "--noserver", "--render", ".", "--outputDir", outputDir)
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr
				command.Dir = filepath.Dir(pathtoserverbin)
				err = command.Run()
				if err != nil {
					fmt.Println("Error: unable to build the ssg pages.")
				} else {
					if verbose {
						fmt.Println("ssg pages built.")
					}
				}

				os.Exit(1)
				return
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
	},
}

func renderPages(renderPath string, renderOutputDir string, releasebuild bool) error {
	pathToServerBinary := filepath.Join(".", "dist", "server", "csr", "tmp", "main")
	if releasebuild {
		pathToServerBinary = filepath.Join(".", "dist", "server", "csr", "release", "main")
	}

	cmd := exec.Command(pathToServerBinary, "--render", renderPath, "--outputDir", renderOutputDir)
	if basepath != "" {
		cmd.Args = append(cmd.Args, "--basepath", basepath)
	}

	// Set working directory to where the binary lives
	// This ensures it can find its source content using relative paths
	cmd.Dir = filepath.Join(".", "dist", "server", "csr", "tmp")
	if releasebuild {
		cmd.Dir = filepath.Join(".", "dist", "server", "csr", "release")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("render failed: %w\noutput: %s", err, output)
	}

	if verbose {
		fmt.Println("render successful")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&basepath, "basepath", "", "/", "base path for the project")
	buildCmd.Flags().BoolVarP(&csr, "csr", "c", false, "build for client-side rendering")
	buildCmd.Flags().BoolVarP(&ssr, "ssr", "s", false, "build for server-side rendering")
	buildCmd.Flags().BoolVarP(&ssg, "ssg", "g", false, "build for static site generation")
	buildCmd.Flags().BoolVarP(&releaseMode, "release", "r", false, "build in release mode")
	buildCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	buildCmd.Flags().BoolVarP(&nohmr, "nohmr", "", false, "disable hot module replacement")
}
