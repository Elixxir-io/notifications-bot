////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Handles command-line version functionality

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/xx_network/primitives/utils"
)

// Change this value to set the version for this build
const currentVersion = "3.0.0"

func printVersion() {
	fmt.Printf("Elixxir Notifications Server v%s -- %s\n\n", SEMVER, GITVERSION)
	fmt.Printf("Dependencies:\n\n%s\n", DEPENDENCIES)
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(generateCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and dependency information for the Elixxir binary",
	Long:  `Print the version and dependency information for the Elixxir binary`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generates version and dependency information for the Elixxir binary",
	Long:  `Generates version and dependency information for the Elixxir binary`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.GenerateVersionFile(currentVersion)
	},
}
