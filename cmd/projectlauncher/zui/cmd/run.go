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

var nohmr, nobuild bool
var releaseMode bool
var port, host string

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run starts an instance of the dev server and serves the client in devmode.",
	Long: `
		Run starts an instance of the dev server and serves the client in devmode.
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
			err = Run(buildtags...)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
				return
			}

			err = SaveConfig()
			if err != nil {
				fmt.Println(err)
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

// Run builds and run an application.
func Run(buildtags ...string) error {
	if On("web") {
		// Run allows the testing of the application in the browser.
		// By default it is a client side rendering application.
		// The server is also built and run in the background since it serves the app locally
		//

		servmod := "csr"
		if ssr {
			servmod = "ssr"
		}
		if ssg {
			servmod = "ssg"
		}

		/*
			if tinygo {
				releaseMode = true
			} else {
				releaseMode = false
			}
		*/

		rmode := "tmp"
		if releaseMode {
			rmode = "release"
		}

		folder := ".root"
		if basepath != "/" {
			folder = filepath.Join(folder, basepath)
		}

		err := Build(true, nil) // TODO add build options, e.g. nohmr should be propagated to client
		if err != nil {
			return err
		}

		if tinygo {
			err = CopyWasmExecJsTinygo(filepath.Join(".", "dist", rmode, "client", folder))
			if err != nil {
				return err
			}
		} else {
			err = CopyWasmExecJs(filepath.Join(".", "dist", rmode, "client", folder))
			if err != nil {
				return err
			}
		}

		serverbinpath := filepath.Join(".", "dist", rmode, "server", "csr", folder, "main")
		if ssr {
			serverbinpath = filepath.Join(".", "dist", rmode, "server", "ssr", folder, "main")
		}
		if ssg {
			serverbinpath = filepath.Join(".", "dist", rmode, "server", "ssr", folder, "main")
		}

		err = Build(false, append(buildtags, "server", servmod))
		if err != nil {
			return err
		}

		if verbose {
			fmt.Println("client and server built.")
		}

		args := []string{"-host", host, "-port", port}
		if nohmr || releaseMode {
			args = append(args, "--nohmr")
		}

		// Let's run the default server.
		cmd := exec.Command(serverbinpath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Dir = filepath.Join(".")
		err = cmd.Run()
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
	if !nohmr && !releaseMode {
		flags[uipkg+"/drivers/js.HMRMode"] = "true"
	} else {
		flags[uipkg+"/drivers/js.HMRMode"] = "false"
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
	runCmd.Flags().BoolVarP(&ssr, "ssr", "s", false, "Runs the server in server-side rendering mode")
	runCmd.Flags().BoolVarP(&ssg, "ssg", "g", false, "Runs the server in static file mode for ssg.")
	runCmd.Flags().BoolVarP(&nohmr, "nohmr", "", false, "Disable hot reloading")
	runCmd.Flags().BoolVarP(&nobuild, "nobuild", "", false, "run the app without rebuilding it")
}
