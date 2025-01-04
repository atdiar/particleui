/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS> TODO DEBUG
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

var tinygo bool

// TODO update the URL for the installers once the project is moved to its final location. (DEBUG)

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "release is the command to build the project for production",
	Long: `

		Release is the command to build the project for production.
		It prepares a new release of the project.
		It builds the source and create an optimized executable.
	
	`,
	Run: func(cmd *cobra.Command, args []string) {
		releaseMode = true
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
			if !csr {
				err := Build(true, nil)
				if err != nil {
					fmt.Println("Error: unable to build the release version of the project.", err)
					os.Exit(1)
					return
				}
				err = Build(false, []string{"server", "csr"})
				if err != nil {
					fmt.Println("Error: unable to build the server.", err)
					os.Exit(1)
					return
				}
				// TODO run the server executable in order to build the index.html file.
				//
			} else if ssr {
				err := Build(true, nil)
				if err != nil {
					fmt.Println("Error: unable to build the release version of the project.", err)
					os.Exit(1)
					return
				}
				err = Build(false, []string{"server", "ssr"})
				if err != nil {
					fmt.Println("Error: unable to build the server.", err)
					os.Exit(1)
					return
				}
				// TODO run the server executable in order to build the index.html file.
				//
			} else if ssg {
				err = Build(false, []string{"server", "ssg"})
				if err != nil {
					fmt.Println("Error: unable to build the ssg server.")
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("ssg server built. Can be found in ../dist/server/ssg/release")
				}

				// Now we need to build the pages by running the server executable
				// at least once.
				// The output files will be found in dist/client/ssg/{basepath | root}/release/
				cmd := exec.Command(filepath.Join(".", "dist", "server", "ssg", "release", "main"), "--noserver")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Dir = filepath.Join(".")
				err = cmd.Run()
				if err != nil {
					fmt.Println("Error: unable to build the ssg pages.")
				} else {
					if verbose {
						fmt.Println("ssg pages built.")
					}
				}

				if releaseMode {

					err = copyDirectory(filepath.Join(".", "dev", "build", "ssg"), filepath.Join(".", "release", "app", "ssg"))
					if err != nil {
						fmt.Println("Error: unable to copy the ssg pages.")
						os.Exit(1)
						return
					}
				}

				os.Exit(1)
				return
			}

			// Now we may want to try and optimize the output file  with binaryen's wasm-opt.
			// First, let's check whether binaryen's wasm-opt is installed.
			if _, err := exec.LookPath("wasm-opt"); err != nil {
				// wasm-opt is not available
				fmt.Println("wasm-opt is not installed.")
			} else {
				// Now we can optimize the wasm file with wasm-opt
				// wasm-opt -Oz -o main.wasm main.wasm

				// First let's get the initial size of the file so that we can compute statistics about file size optimization later.
				// We can use the stat command to get the size of the file.
				// stat -c %s main.wasm
				// The output of the command will be the size of the file in bytes.
				var size int64
				cmd := exec.Command("stat", "-c", "%s", filepath.Join(".", "release", "app", basepath, "main.wasm"))
				out, err := cmd.Output()
				if err != nil {
					fmt.Println("Error: unable to get the size of the wasm file.")
					os.Exit(1)
					return
				}
				fmt.Sscanf(string(out), "%d", &size)

				cmd = exec.Command("wasm-opt", "-Oz", "-o", filepath.Join(".", "release", "app", basepath, "main.wasm"), filepath.Join(".", "release", "app", basepath, "main.wasm"))
				err = cmd.Run()
				if err != nil {
					fmt.Println("Error: unable to optimize the wasm file.") // TODO tinygo works but when using the default gc it errors out. // DEBUG
					os.Exit(1)
					return
				}

				if verbose {
					fmt.Println("wasm file optimized.")
				}

				var endsize int64
				cmd = exec.Command("stat", "-c", "%s", filepath.Join(".", "release", "app", basepath, "main.wasm"))
				out, err = cmd.Output()
				if err != nil {
					fmt.Println("Error: unable to get the size of the wasm file.")
					os.Exit(1)
					return
				}
				fmt.Sscanf(string(out), "%d", &endsize)

				if verbose {
					fmt.Printf("wasm file size before optimization: %d bytes\n", size)
					fmt.Printf("wasm file size after optimization: %d bytes\n", endsize)
					fmt.Printf("Optimization ratio: %.2f%%\n", (float64(size)-float64(endsize))/float64(size)*100)
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

		// if binaryen wasm-opt is present, use it to optimize the wasm file (TODO)
		// On project init check if wasm-opt is present, set a flag on the project manifest
		// to notify that it can be optimized with wasm-opt
	},
}

func copyDirectory(src string, dst string) error {
	// Get properties of source directory
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error getting source directory info: %v", err)
	}

	// Create destination directory with same permissions
	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("error creating destination directory: %v", err)
	}

	// Get directory contents
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error reading source directory: %v", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			err = copyDirectory(srcPath, dstPath)
			if err != nil {
				return fmt.Errorf("error copying subdirectory: %v", err)
			}
		} else {
			// Copy file
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return fmt.Errorf("error copying file: %v", err)
			}
		}
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
