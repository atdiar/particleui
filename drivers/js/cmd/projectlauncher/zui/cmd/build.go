/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"net/url"
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
			b,err:= url.Parse(basepath)
			if err != nil{
				fmt.Println("invalid basepath")
				os.Exit(1)
				return
			}
			if b.Path != "/"{
				if b.Path[0] != '/'{
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
				err = Build(filepath.Join(".", "dev", "build", "app", "main.wasm"), nil)
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
					// The output file should be in dev/build/server/csr/
					err = Build(filepath.Join(".", "dev", "build", "server", "csr", "main"), []string{"server", "csr"})
					if err != nil {
						fmt.Println("Error: unable to build the default server.")
						os.Exit(1)
						return
					}
					if verbose {
						fmt.Println("default server built.")
					}
				}

			} else if ssr {
				err = Build(filepath.Join(".", "dev", "build", "app", "main.wasm"), nil)
				if err != nil {
					fmt.Println("Error: unable to build the default app.")
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("wasm app built.")
				}

				// Let's build the default server.
				// The output file should be in dev/build/server/ssr/
				err = Build(filepath.Join(".", "dev", "build", "server", "ssr", "main"), []string{"server", "ssr"})
				if err != nil {
					fmt.Println("Error: unable to build the ssr server.")
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("ssr server built.")
				}
			} else if ssg {
				err = Build(filepath.Join(".", "dev", "build", "server", "ssg", "main"), []string{"server", "ssg"})
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
				// The output files will be found in dev/build/ssg/static
				cmd := exec.Command(filepath.Join(".", "dev", "build", "server", "ssg", "main"), "noserver")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Dir = filepath.Join(".", "dev", "build", "server", "ssg")
				err = cmd.Run()
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

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&basepath, "basepath", "", "/", "base path for the project")
	buildCmd.Flags().BoolVarP(&csr, "csr", "c", false, "build for client-side rendering")
	buildCmd.Flags().BoolVarP(&ssr, "ssr", "s", false, "build for server-side rendering")
	buildCmd.Flags().BoolVarP(&ssg, "ssg", "g", false, "build for static site generation")
}
