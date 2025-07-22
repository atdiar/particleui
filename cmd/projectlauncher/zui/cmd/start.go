/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

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
		Start()
	},
}

// Start builds and start the application server.
func Start() {
	err := LoadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}

	if On("web") {
		err = runFunc()
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
}
