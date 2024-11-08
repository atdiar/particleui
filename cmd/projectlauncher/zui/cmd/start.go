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
		For the web target, the apps is served at locahost:8888 by default with hot reloading
		enabled. To disbale hoitreloading, use the --nohmr flag.
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
		// Start allows the testing of the application in the browser.
		// By default it is a client side rendering application.
		// The server is also built and start in the background since it serves the app locally
		//

		servmod := "csr"
		if ssr {
			servmod = "ssr"
		}
		if ssg {
			servmod = "ssg"
		}

		basepathseg := strings.TrimSuffix(basepath, "/")
		basepathseg = strings.TrimPrefix(basepathseg, "/")

		serverbinpath := filepath.Join(".", "dev", "build", "server", "csr", basepathseg, "main")
		if ssr {
			serverbinpath = filepath.Join(".", "dev", "build", "server", "ssr", basepathseg, "main")
		}
		if ssg {
			serverbinpath = filepath.Join(".", "dev", "build", "server", "ssr", basepathseg, "main")
		}

		err := Build(serverbinpath, append(buildtags, "server", servmod))
		if err != nil {
			return err
		}

		if verbose {
			fmt.Println("client and server built.")
		}

		args := []string{"-host", host, "-port", port}
		if nohmr {
			args = append(args, "--nohmr")
		}

		// Let's start the default server.
		cmd := exec.Command(serverbinpath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Dir = filepath.Join(".")
		err = cmd.Start()
		if err != nil {
			return err
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
	startCmd.Flags().BoolVarP(&ssr, "ssr", "s", false, "Starts the server in server-side rendering mode")
	startCmd.Flags().BoolVarP(&ssg, "ssg", "g", false, "Starts the server in static file mode for ssg.")
	startCmd.Flags().BoolVarP(&nohmr, "nohmr", "", false, "Disable hot reloading")
	startCmd.Flags().BoolVarP(&nobuild, "nobuild", "", false, "run the app without rebuilding it")
}
