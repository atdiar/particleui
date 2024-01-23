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


