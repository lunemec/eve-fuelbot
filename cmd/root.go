package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "eve-fuelbot",
	Short: "Discord bot for structure fuel notification",
	Long:  ``,
}

const (
	addr           = "localhost:3000"
	eveCallbackURL = "http://localhost:3000/callback"
)

// variables parsed from CLI.
var (
	authfile     string // path to file with authentication data
	sessionKey   string // session key used for user session encryption
	eveClientID  string // EVE APP Client ID
	eveSSOSecret string // EVE APP SSO secret
)

var eveScopes = []string{"publicData", "esi-corporations.read_structures.v1", "esi-universe.read_structures.v1"}

func httpClient() *http.Client {
	transport := httpcache.NewTransport(httpcache.NewMemoryCache())
	transport.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	client := http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
	return &client
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
