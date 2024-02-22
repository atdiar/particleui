/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var release bool

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("release called")

		// TODO when building wasm remove debug info
		// And any other potential optimization

		// Add -tinygo flag to build with the tu=inygo compiler if present

		// remove debug info from wasm binary by using ldlflags="-s -w" (TODO)

		// if bynaryen wasm-opt is present, use it to optimize the wasm file (TODO)
		// On project init check if wasm-opt is present, set a flag on the project manifest
		// to notify that it can be optimnized with wasm-opt
		// otherwise, there should be an option to install it from our mirror
	},
}

func init() {
	rootCmd.AddCommand(releaseCmd)

	releaseCmd.Flags().BoolVarP(&release,"release","",false, "builds a production version of the project.")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// releaseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// releaseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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