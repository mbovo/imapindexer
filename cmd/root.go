/*
			Copyright © 2023 Manuel Bovo <manuel.bovo@gmail.com>

	    This program is free software: you can redistribute it and/or modify
	    it under the terms of the GNU General Public License as published by
	    the Free Software Foundation, either version 3 of the License, or
	    (at your option) any later version.

	    This program is distributed in the hope that it will be useful,
	    but WITHOUT ANY WARRANTY; without even the implied warranty of
	    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	    GNU General Public License for more details.

	    You should have received a copy of the GNU General Public License
	    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "imapindexer",
	Short: "Read all the mailboxes of a imap account and index the messages on a ZincSearch instance",
	Long: `imapindexer  Copyright (C) 2023  Manuel Bovo
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.
See <https://www.gnu.org/licenses/>.

Index all emails messages of your IMAP mailboxes and store 
them into a ZincSearch/Elasticsearch to made them fully searchable.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.imapindexer.yaml)")
	rootCmd.Flags().String("imap.address", "", "IMAP server address")
	rootCmd.Flags().String("imap.username", "", "IMAP username")
	rootCmd.Flags().String("imap.password", "", "IMAP password")
	rootCmd.Flags().String("imap.mailbox", "INBOX", "IMAP mailbox pattern")
	rootCmd.Flags().String("zinc.address", "", "ZincSearch server address")
	rootCmd.Flags().String("zinc.username", "", "ZincSearch username")
	rootCmd.Flags().String("zinc.password", "", "ZincSearch password")
	rootCmd.Flags().String("zinc.index", "mail_index", "ZincSearch index name")
	rootCmd.Flags().Int("indexer.workers", 1, "Number of imap workers to use")
	rootCmd.Flags().Int("indexer.buffer", 100, "Size of buffer for messages channel")
	rootCmd.Flags().Int("indexer.batch", 100, "Number of message to send to ZincSearch in a single batch")

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Logger().Level(zerolog.InfoLevel)

	viper.BindPFlags(rootCmd.PersistentFlags())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".imapindexer" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".imapindexer")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
