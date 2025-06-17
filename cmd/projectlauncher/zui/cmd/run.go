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

var uipkg = "github.com/atdiar/particleui"

var nolr, nobuild bool
var releaseMode bool
var port, host string

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run starts an instance of the dev server and serves the client in devmode.",
	Long: `
		Run starts an instance of the dev server and serves the client in devmode.
		For the web target, the apps is served at locahost:8888 by default with live reloading
		enabled. To disbale hoitreloading, use the --nolr flag.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		err := LoadConfig()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}

		if On("web") {

			err = BuildAndRun(args...)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
				return
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

// Run builds and run an application.
func BuildAndRun(args ...string) error {
	if On("web") {
		// Run launches the webserver used to serve the application.
		// after having built the client and the server.

		// 1. we run the build command with the same arguments as the run command.
		buildcmd := exec.Command("zui", append([]string{"build"}, args...)...)
		buildcmd.Stdout = os.Stdout
		buildcmd.Stderr = os.Stderr
		err := buildcmd.Run()
		if err != nil {
			return fmt.Errorf("error building the application: %w", err)
		}
		if verbose {
			fmt.Println("Now let's try running the server...")
		}

		// 2. we run the server command with the same arguments as the run command.

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

func ldflags() string {
	flags := make(map[string]string)

	if !releaseMode {
		flags[uipkg+"/drivers/js.DevMode"] = "true"
	}
	if ssr {
		flags[uipkg+"/drivers/js.SSRMode"] = "true"
	}
	if ssg {
		flags[uipkg+"/drivers/js.SSGMode"] = "true"
	}
	if !nolr && !releaseMode {
		flags[uipkg+"/drivers/js.LRMode"] = "true"
	} else {
		flags[uipkg+"/drivers/js.LRMode"] = "false"
	}

	if basepath != "/" {
		flags[uipkg+"/drivers/js.BasePath"] = basepath
	}

	var ldflags = []string{}
	for key, value := range flags {
		ldflags = append(ldflags, fmt.Sprintf("-X %s=%s", key, value))
	}
	return strings.Join(ldflags, " ")
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&basepath, "basepath", "", "/", "base path for the project")
	runCmd.Flags().StringVarP(&host, "host", "", "localhost", "Host name for the server")
	runCmd.Flags().StringVarP(&port, "port", "p", "8888", "Port number for the server")
	runCmd.Flags().BoolVarP(&releaseMode, "release", "r", false, "Run in release mode")
	runCmd.Flags().BoolVarP(&csr, "csr", "c", false, "build for client-side rendering")
	runCmd.Flags().BoolVarP(&ssr, "ssr", "s", false, "Runs the server in server-side rendering mode")
	runCmd.Flags().BoolVarP(&ssg, "ssg", "g", false, "Runs the server in static file mode for ssg.")
	runCmd.Flags().BoolVarP(&nolr, "nolr", "", false, "Disable live reloading")
}
