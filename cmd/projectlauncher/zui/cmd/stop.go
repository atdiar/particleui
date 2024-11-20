/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the running dev server",
	Run: func(cmd *cobra.Command, args []string) {
		err := LoadConfig()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}

		if On("web") {
			// Get the port number
			port, ok := config["port"]
			if !ok {
				if verbose {
					fmt.Println("No port number found in config file")
				}
				os.Exit(0)
			}

			host, ok := config["host"]
			if !ok {
				if verbose {
					fmt.Println("No host found in config file")
				}
				os.Exit(0)
			}

			serverAddress := fmt.Sprintf("http://%s:%s/stop", host, port)

			// Send shutdown request to the server
			_, err = http.Get(serverAddress)
			if err != nil {
				log.Fatalf("Failed to send shutdown request: %v", err)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// stopCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// stopCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
