/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start starts an instance of the dev server and serves the client in devmode.",
	Long: `
		start starts an instance of the dev server after having built it.
		For the web target, the apps is served at locahost:8888 by default with live reloading
		enabled. To disbale live reloading, use the --nolr flag.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		err := LoadConfig()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}

		if On("web") {

			var buildtags = []string{}

			var csr = true
			if ssr {
				csr = false
				buildtags = append(buildtags, "ssr")
			}

			if ssg {
				csr = false
				buildtags = append(buildtags, "ssg")
			}

			if csr {
				buildtags = append(buildtags, "csr")
			}

			// Let's build the app.
			err = Start(buildtags...)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
				return
			}

			if verbose {
				fmt.Println("server running on port ", port)
			}

		} else if On("mobile") {
			// TODO
			fmt.Println("building for mobile is not yet supported")
			os.Exit(1)
		} else if On("desktop") {
			// TODO
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

// Start builds and start the application server.
func Start(buildtags ...string) error {
	if On("web") {
		// Start launched the webserver that serves the application.
		// The application client and server are expected to have been built first.

		rootdirectory := "_root"
		if basepath != "/" {
			rootdirectory = basepath
		}

		var outputPath string
		if csr {
			outputPath = getServerBinaryPath("csr", releaseMode, rootdirectory)
		} else if ssr {
			outputPath = getServerBinaryPath("ssr", releaseMode, rootdirectory)
		} else if ssg {
			outputPath = getServerBinaryPath("ssg", releaseMode, rootdirectory)
		}

		args := []string{"-host", host, "-port", port}
		if nolr || releaseMode {
			args = append(args, "--nolr")
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current working directory: %w", err)
		}
		absoluteBinaryPath := filepath.Join(cwd, outputPath)
		if _, err := os.Stat(absoluteBinaryPath); os.IsNotExist(err) {
			return fmt.Errorf("server binary does not exist at path: %s", absoluteBinaryPath)
		}

		// Let's run the default server.
		cmd := exec.Command(absoluteBinaryPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Dir = filepath.Dir(absoluteBinaryPath)
		err = cmd.Run()
		if err != nil {
			return err
		}

		if verbose {
			// Print the exact command
			fmt.Printf("Running command: %s %s\n", absoluteBinaryPath, strings.Join(args, " "))
			// Print the ldglags
			fmt.Printf("Using ldflags: %s\n", ldflags())
			// Print the server URL
			fmt.Printf("Running server at http://%s:%s with base path %s ...\n", host, port, basepath)
		}

		return nil
	}

	if On("terminal") {
		return fmt.Errorf("building for terminal is not yet supported")
	}

	if On("mobile") {
		return fmt.Errorf("building for mobile is not yet supported")
	}

	if On("desktop") {
		return fmt.Errorf("building for desktop is not yet supported")
	}

	return fmt.Errorf("unknown platform")

}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&basepath, "basepath", "", "/", "base path for the project")
	startCmd.Flags().StringVarP(&host, "host", "", "localhost", "Host name for the server")
	startCmd.Flags().StringVarP(&port, "port", "p", "8888", "Port number for the server")
	startCmd.Flags().BoolVarP(&releaseMode, "release", "r", false, "Start in release mode")
	startCmd.Flags().BoolVarP(&csr, "csr", "c", false, "Starts the server in client-side rendering mode")
	startCmd.Flags().BoolVarP(&ssr, "ssr", "s", false, "Starts the server in server-side rendering mode")
	startCmd.Flags().BoolVarP(&ssg, "ssg", "g", false, "Starts the server in static file mode for ssg.")
	startCmd.Flags().BoolVarP(&nolr, "nolr", "", false, "Disable live reloading")
	startCmd.Flags().BoolVarP(&nobuild, "nobuild", "", false, "run the app without rebuilding it")
}
