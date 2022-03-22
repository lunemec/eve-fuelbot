package cmd

import (
	"fmt"
	"time"

	"github.com/lunemec/eve-fuelbot/pkg/bot"
	"github.com/lunemec/eve-fuelbot/pkg/token"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the discord bot",
	Run:   runBot,
}

var (
	checkInterval      time.Duration
	notifyInterval     time.Duration
	refuelNotification time.Duration

	discordChannelID string
	discordAuthToken string
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&authfile, "auth_file", "a", "auth.bin", "path to file where to save authentication data")
	runCmd.Flags().StringVarP(&sessionKey, "session_key", "s", "", "session key, use random string")
	runCmd.Flags().StringVar(&eveClientID, "eve_client_id", "", "EVE APP client id")
	runCmd.Flags().StringVar(&eveSSOSecret, "eve_sso_secret", "", "EVE APP SSO secret")
	runCmd.Flags().StringVar(&discordChannelID, "discord_channel_id", "", "ID of discord channel")
	runCmd.Flags().StringVar(&discordAuthToken, "discord_auth_token", "", "Auth token for discord")
	runCmd.Flags().DurationVar(&checkInterval, "check_interval", 1*time.Hour, "how often to check EVE ESI API (default 1H)")
	runCmd.Flags().DurationVar(&notifyInterval, "notify_interval", 12*time.Hour, "how often to spam discord (default 12H)")
	runCmd.Flags().DurationVar(&refuelNotification, "refuel_notification", 5*24*time.Hour, "how far in advance would you like to be notified about the fuel (default 5 days)")

	must(runCmd.MarkFlagRequired("session_key"))
	must(runCmd.MarkFlagRequired("eve_client_id"))
	must(runCmd.MarkFlagRequired("eve_sso_secret"))
	must(runCmd.MarkFlagRequired("discord_channel_id"))
	must(runCmd.MarkFlagRequired("discord_auth_token"))
}

func runBot(cmd *cobra.Command, args []string) {
	fastLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("error inicializing logger: %s", err))
	}
	log := fastLog.Sugar()

	client := httpClient()

	tokenStorage := token.NewFileStorage(authfile)
	tokenSource := token.NewSource(log, client, tokenStorage, []byte(sessionKey), eveClientID, eveSSOSecret, eveCallbackURL, eveScopes)

	discord, err := discordgo.New("Bot " + discordAuthToken)
	if err != nil {
		panic(fmt.Sprintf("error inicializing discord client: %s", err))
	}
	bot := bot.NewFuelBot(log, client, tokenSource, discord, discordChannelID, checkInterval, notifyInterval, refuelNotification)
	err = bot.Bot()
	// systemd handles reload, so we can panic on error.
	if err != nil {
		panic(err)
	}
}
