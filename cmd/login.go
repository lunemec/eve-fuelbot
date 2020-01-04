package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lunemec/eve-fuelbot/pkg/handler"
	"github.com/lunemec/eve-fuelbot/pkg/token"

	"github.com/braintree/manners"
	open "github.com/pbnj/go-open"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with EVE SSO and save token to be used by the bot",
	Run:   runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&authfile, "auth_file", "a", "auth.bin", "path to file where to save authentication data")
	loginCmd.Flags().StringVarP(&sessionKey, "session_key", "s", "", "session key, use random string")
	loginCmd.Flags().StringVar(&eveClientID, "eve_client_id", "", "EVE APP client id")
	loginCmd.Flags().StringVar(&eveSSOSecret, "eve_sso_secret", "", "EVE APP SSO secret")

	loginCmd.MarkFlagRequired("session_key")
	loginCmd.MarkFlagRequired("eve_client_id")
	loginCmd.MarkFlagRequired("eve_sso_secret")
}

func runLogin(cmd *cobra.Command, args []string) {
	fastLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("error inicializing logger: %v", err))
	}
	log := fastLog.Sugar()
	signalChan := make(chan os.Signal, 1)
	// Notify signalChan on SIGINT and SIGTERM.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	if _, err := os.Stat(authfile); !os.IsNotExist(err) {
		err = os.Remove(authfile)
		if err != nil {
			panic(fmt.Sprintf("unable to delete file: %s please remove it by hand", authfile))
		}
	}

	handler := handler.New(signalChan, log, httpClient(), token.NewFileStorage(authfile), []byte(sessionKey), eveClientID, eveSSOSecret, eveCallbackURL, eveScopes)
	server := manners.NewWithServer(&http.Server{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      handler,
	})

	go func() {
		for s := range signalChan {
			log.Infof("Captured %v. Exiting...", s)
			server.Close()
		}
	}()

	// Open default web browser after 1s.
	go func() {
		openAddr := fmt.Sprintf("http://%s", addr)
		time.Sleep(1 * time.Second)
		log.Infof("Opening browser at %s", openAddr)
		open.Open(openAddr)
	}()

	log.Infof("Listening on %v", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Errorf("ListenAndServe error: %v", err)
	}
}
