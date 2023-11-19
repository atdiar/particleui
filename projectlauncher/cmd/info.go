/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "info prints the project information.",
	Run: func(cmd *cobra.Command, args []string) {
		err := LoadConfig()
		if err != nil{
			fmt.Println("Unable to retrieve project info.Couldn't load "+ configFileName)
			os.Exit(1)
		}
		j,err:= json.MarshalIndent(config,"","  ")
		if err != nil{
			fmt.Println("Error occured while marshalling project info.")
			os.Exit(1)
		}
		fmt.Println(string(j))
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// infoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// infoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
