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

var nohmr bool
var releaseMode bool

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run starts an instance oif the dev server.",
	Long: `
		Run starts an instance of the dev server.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		err:= LoadConfig()
			if err != nil{
				fmt.Println(err)
				os.Exit(1)
				return
			}

			if On("web"){
				
				if !(ssr || ssg){

					// Let's build and run the app in csr mode.
					var buildOptions =  []string{"csr"}
					if !releaseMode{
						buildOptions = append(buildOptions, "dev")
					}
					if nohmr{
						buildOptions = append(buildOptions, "nohmr")
					}

					var host = "localhost"
					if host, err := cmd.Flags().GetString("host"); err == nil {
						buildOptions = append(buildOptions, fmt.Sprintf("host=%s", host))
					}

					var port = "8888"
					if port, err := cmd.Flags().GetString("port"); err == nil {
						buildOptions = append(buildOptions, fmt.Sprintf("port=%s", port))
					}

					// Let's build the app.
					err = Run(buildOptions...)
					if err != nil {
						fmt.Println(err.Error())
						os.Exit(1)
						return
					}

					config["port"]=port
					config["host"]=host
					err = SaveConfig()
					if err != nil{
						fmt.Println(err)
						os.Exit(1)
						return
					}

					if verbose{
						fmt.Println("default server running...")
					}
				} else if ssr{

					var buildOptions =  []string{"ssr"}
					if !releaseMode{
						buildOptions = append(buildOptions, "dev")
					}
					if nohmr{
						buildOptions = append(buildOptions, "nohmr")
					}

					var host = "localhost"
					if host, err := cmd.Flags().GetString("host"); err == nil {
						buildOptions = append(buildOptions, fmt.Sprintf("host=%s", host))
					}

					var port = "8888"
					if port, err := cmd.Flags().GetString("port"); err == nil {
						buildOptions = append(buildOptions, fmt.Sprintf("port=%s", port))
					}

					// Let's run the default server.
					// The output file should be in dev/build/server/ssr/
					err = Run(buildOptions...)
					if err != nil {
						fmt.Println("Error: unable to build the ssr server.")
						os.Exit(1)
						return
					}

					config["port"]=port
					config["host"]=host
					err = SaveConfig()
					if err != nil{
						fmt.Println(err)
						os.Exit(1)
						return
					}

					if verbose{
						fmt.Println("ssr server running on port "+port+"...")
					}
				} else if ssg{
					// TODO
					fmt.Println("building for ssg is not yet supported")
					os.Exit(1)
					return
				}
			} else if On("mobile"){
				// TODO
				fmt.Println("building for mobile is not yet supported")
				os.Exit(1)
			} else if On("desktop"){
				// TODO
				fmt.Println("building for desktop is not yet supported")
				os.Exit(1)
			} else if On("terminal"){
				// TODO
			} else{
				fmt.Println("unknown platform")
				os.Exit(1)
				return
			}
	},
}

// Run builds and run an application.
func Run(buildoptions ...string) error{
	if On("web"){

		if !(ssr || ssg){
			csr = true
			buildoptions = append(buildoptions, "csr")
		}
	
		if csr{
			err := Build(filepath.Join(".","dev","build","app", "main.wasm"), nil)
			if err != nil {
				return err
			}

			serverbinpath := filepath.Join(".","dev","build","server", "csr","main")
			if releaseMode{
				buildoptions = append(buildoptions, "release")
			}
			if nohmr{
				buildoptions = append(buildoptions, "nohmr")
			}
			err = Build(serverbinpath,buildoptions)
			if err != nil {
				return err
			}

			if verbose{
				fmt.Println("client and server built.")
			}

			// Let's run the default server.
			cmd:= exec.Command(serverbinpath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = filepath.Join(".","dev","build","server", "csr")
			err = cmd.Run()
			if err != nil {
				return err
			}

			return nil
		} else if ssr{
			err := Build(filepath.Join(".","dev","build","app", "main.wasm"),nil)
			if err != nil {
				return err
			}

			serverbinpath := filepath.Join(".","dev","build","server", "ssr","main")
			if releaseMode{
				buildoptions = append(buildoptions, "release")
			}
			if nohmr{
				buildoptions = append(buildoptions, "nohmr")
			}
			err = Build(serverbinpath,buildoptions)
			if err != nil {
				return err
			}

			if verbose{
				fmt.Println("wasm app built.")
			}

			// Let's build the default server.
			// The output file should be in dev/build/server/ssr/
			cmd:= exec.Command(serverbinpath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = filepath.Join(".","dev","build","server", "ssr")
			err = cmd.Run()
			if err != nil {
				return err
			}

			return nil
		} else if ssg{
			// TODO
			fmt.Println("building for ssg is not yet supported")
			os.Exit(1)
			return nil
		}
	
		
		return nil
	}

	if On("terminal"){
		return fmt.Errorf("building for terminal is not yet supported")
	}

	if On("mobile"){
		return fmt.Errorf("building for mobile is not yet supported")
	}

	if On("desktop"){
		return fmt.Errorf("building for desktop is not yet supported")
	}

	return fmt.Errorf("unknown platform")

}

func ldflags() string {
	flags := make(map[string]string)

	if releaseMode {
		flags[uipkg + "/drivers/js.DevMode"] = "true"
	}
	if ssr {
		flags[uipkg + "/drivers/js.SSRMode"] = "true"
	}
	if ssg {
		flags[uipkg + "/drivers/js.SSGMode"] = "true"
	}
	if nohmr {
		flags[uipkg + "/drivers/js.HMRMode"] = "true"
	}

	var ldflags []string
	for key, value := range flags {
		ldflags = append(ldflags, fmt.Sprintf("-X %s=%s", key, value))
	}
	return strings.Join(ldflags, " ")
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().String("host", "localhost", "Host name for the server")
	runCmd.Flags().String("port", "8888", "Port number for the server")
	runCmd.Flags().BoolVarP(&releaseMode, "release", "r", false, "Run in release mode")
    runCmd.Flags().BoolVarP(&ssr, "ssr", "s", false, "Runs the server in server-side rendering mode")
    runCmd.Flags().BoolVarP(&ssg, "ssg", "g", false, "Runs the server in static file mode for ssg.")
    runCmd.Flags().BoolVarP(&nohmr, "nohmr", "nh", false, "Disable hot module replacement")
}
