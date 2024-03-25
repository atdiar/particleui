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

var tinygo bool
var release bool

var tinygoinstallerURL string = "github.com/atdiar/particleui/drivers/js/cmd/tinygoinstall"
var binaryeninstallerURL string = "github.com/atdiar/particleui/drivers/js/cmd/binaryeninstall"

// TODO update the URL for the installers once the project is moved to its final location. (DEBUG)

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "release is the command to build the project for production",
	Long: `

		Release is the command to build the project for production.
		It prepares a new release of the project.
		It builds the source and create an optimized executable.
		(TBC) // TODO
	
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

			// Now we may want to try and optimize the output file  with binaryen's wasm-opt.
			// First, let's check whether binaryen's wasm-opt is installed.
			// If not, let's install it:
			if _, err := exec.LookPath("wasm-opt"); err != nil {
				// wasm-opt is not available
				if err := installBinaryenGo(verbose); err != nil {
					fmt.Println(err)
					os.Exit(1)
					return
				}
			}

			// Now we can optimize the wasm file with wasm-opt
			// wasm-opt -Oz -o main.wasm main.wasm
			cmd := exec.Command("wasm-opt", "-Oz", "-o", filepath.Join(".", "dev", "build", "app", "main.wasm"), filepath.Join(".", "dev", "build", "app", "main.wasm"))
			err = cmd.Run()
			if err != nil {
				fmt.Println("Error: unable to optimize the wasm file.")
				os.Exit(1)
				return
			}

			if verbose {
				fmt.Println("wasm file optimized.")
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

		// TODO when building wasm remove debug info
		// And any other potential optimization

		// Add -tinygo flag to build with the tinygo compiler if present

		// remove debug info from wasm binary by using ldlflags="-s -w" (TODO)

		// if bynaryen wasm-opt is present, use it to optimize the wasm file (TODO)
		// On project init check if wasm-opt is present, set a flag on the project manifest
		// to notify that it can be optimnized with wasm-opt
		// otherwise, there should be an option to install it from our mirror
	},
}

func installBinaryenGo(verbosity bool) error {
	var response string
	fmt.Print("Would you like to install the binaryen toolchain? (y/n): ")
	_, err := fmt.Scan(&response)
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	if response == "y" {
		// Prepare the command for installing binaryen
		installCmdArgs := []string{"install"}
		if verbosity {
			// Append the verbosity flag based on the verbosity argument
			installCmdArgs = append(installCmdArgs, "-verbose")
		}
		installCmdArgs = append(installCmdArgs, binaryeninstallerURL)
		cmd := exec.Command("go", installCmdArgs...)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error installing binaryen: %v", err)
		}

		// Prepare the command to run the binaryen installer, assuming it also accepts a verbosity flag
		binaryenInstallCmd := []string{"binaryeninstall"}
		if verbosity {
			// Append the verbosity flag based on the verbosity argument
			binaryenInstallCmd = append(binaryenInstallCmd, "-verbose")
		}
		cmd = exec.Command(binaryenInstallCmd[0], binaryenInstallCmd[1:]...)
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("error running binaryen installer: %v", err)
		}
	} else {
		return fmt.Errorf("binaryen toolchain installation aborted")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(releaseCmd)

	releaseCmd.Flags().StringVarP(&basepath, "basepath", "", "/", "base path for the project")
	releaseCmd.Flags().BoolVarP(&csr, "csr", "c", false, "build for client-side rendering")
	releaseCmd.Flags().BoolVarP(&ssr, "ssr", "s", false, "build for server-side rendering")
	releaseCmd.Flags().BoolVarP(&ssg, "ssg", "g", false, "build for static site generation")

	releaseCmd.Flags().BoolVarP(&tinygo, "tinygo", "", false, "build with the tinygo toolchain.")
}


/*

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// promptUser prompts the user with a question and returns true for "y" and false for "n".
// It repeats the prompt until a valid input is received.
func promptUser(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(question)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading response. Please try again.")
			continue
		}
		response = strings.TrimSpace(response)
		if response == "y" {
			return true
		} else if response == "n" {
			return false
		} else {
			fmt.Println("Invalid input. Please answer 'y' or 'n'.")
		}
	}
}

// checkAndInstallWasmOpt checks if wasm-opt is installed.
// If not, it prompts the user to install binaryen, ensuring at least version 116.
func checkAndInstallWasmOpt(installerPkg string, minVersion int) bool {
	// Check if wasm-opt is available
	if _, err := exec.LookPath("wasm-opt"); err == nil {
		fmt.Println("wasm-opt is already installed.")
		return true
	}

	fmt.Println("wasm-opt is not available.")
	if promptUser("Do you want to install binaryen to get wasm-opt? (y/n): ") {
		// User agreed to install
		return installBinaryen(installerPkg, minVersion)
	}

	// wasm-opt not installed or user chose not to install
	fmt.Println("Installation aborted.")
	return false
}

// installBinaryen uses go install to install the specified Go package as an installer,
// then runs the installed binary with the version as an argument.
func installBinaryen(installerPkg string, version int) bool {
	// Ensure the version is at least 116
	if version < 116 {
		version = 116
	}

	// Install the Go package (installer)
	cmdGet := exec.Command("go", "install", fmt.Sprintf("%s@latest", installerPkg))
	if err := cmdGet.Run(); err != nil {
		fmt.Printf("Failed to install the installer: %v\n", err)
		return false
	}

	// Extract the binary name from the package path
	installerName := installerPkg[strings.LastIndex(installerPkg, "/")+1:]
	// Assuming the GOPATH/bin or GOBIN is in PATH, execute the installer with the version argument
	cmdRun := exec.Command(installerName, fmt.Sprintf("%d", version))
	if err := cmdRun.Run(); err != nil {
		fmt.Printf("Failed to execute the installer with version argument: %v\n", err)
		return false
	}

	fmt.Println("binaryen installation completed.")
	return true
}

func main() {
	installerPkg := "github.com/your/repo/path" // Replace with the actual Go package path
	minVersion := 116 // The minimum required version of binaryen
	if checkAndInstallWasmOpt(installerPkg, minVersion) {
		fmt.Println("wasm-opt is ready to use.")
	} else {
		fmt.Println("wasm-opt is not available.")
	}
}




*/