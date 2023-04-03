package bot

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lunemec/eve-fuelbot/pkg/token"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

// Bot what a bot does.
type Bot interface {
	Bot() error
}

type fuelBot struct {
	tokenSource token.Source
	log         logger
	esi         *goesi.APIClient
	discord     *discordgo.Session
	channelID   string

	httpClient *http.Client

	checkInterval      time.Duration
	notifyInterval     time.Duration
	refuelNotification time.Duration

	notified map[int64]time.Time
}

type logger interface {
	Infow(string, ...interface{})
	Errorw(string, ...interface{})
}

type structureData struct {
	CorporationData esi.GetCorporationsCorporationIdStructures200Ok
	UniverseData    esi.GetUniverseStructuresStructureIdOk
}

// NewFuelBot returns new bot instance.
func NewFuelBot(log logger, client *http.Client, tokenSource token.Source, discord *discordgo.Session, channelID string, checkInterval, notifyInterval, refuelNotification time.Duration) Bot {
	log.Infow("EVE FuelBot starting",
		"check_interval", checkInterval,
		"notify_interval", notifyInterval,
		"refuel_notification", refuelNotification,
	)
	esi := goesi.NewAPIClient(client, "EVE FuelBot")
	return &fuelBot{
		tokenSource:        tokenSource,
		log:                log,
		esi:                esi,
		discord:            discord,
		channelID:          channelID,
		httpClient:         &http.Client{Timeout: 5 * time.Second},
		checkInterval:      checkInterval,
		notifyInterval:     notifyInterval,
		refuelNotification: refuelNotification,
		notified:           make(map[int64]time.Time),
	}
}

// Bot - you know, do what a bot does.
func (b *fuelBot) Bot() error {
	err := b.discord.Open()
	if err != nil {
		return errors.Wrap(err, "unable to connect to discord")
	}
	// Add handler to listen for "!fuel" messages to report all structures fuel
	// expiration date.
	b.discord.AddHandler(b.messageFuelHandler)

	for {
		structs, err := b.loadStructures()
		if err != nil {
			// Log but do not return error, we don't want to crash on panic.
			b.log.Errorw("Error loading structures",
				"error", errors.Wrap(err, "error loading structure information"),
			)
		}

		// In case of previous error, we are iterating 0 times over nil slice.
		for _, structure := range structs {
			notify := b.shouldNotify(structure)
			if notify {
				b.log.Infow("Sending message",
					"channel_id", b.channelID,
					"structure_id", structure.CorporationData.StructureId,
					"structure_name", structure.UniverseData.Name,
				)
				_, err = b.discord.ChannelMessageSendEmbed(b.channelID, b.message(&structure))
				switch {
				case err != nil:
					b.log.Errorw("Error sending discord message",
						"error", errors.Wrap(err, "error sending discord message"),
					)
					// In case of error, we fall through to the time.Sleep
					// block. We also do not set the structure as notified
					// and it get picked up on next iteration.
					continue
				case err == nil:
					b.setWasNotified(structure)
				}
			}
		}

		time.Sleep(b.checkInterval)
	}
}

func (b *fuelBot) message(structure *structureData) *discordgo.MessageEmbed {
	whereMsg := "`%s`"
	whereMsg = fmt.Sprintf(whereMsg, structure.UniverseData.Name)

	whenMsg := fmt.Sprintf("`%s` (%s)",
		humanize.Time(structure.CorporationData.FuelExpires),
		structure.CorporationData.FuelExpires,
	)

	return &discordgo.MessageEmbed{
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://i.imgur.com/pKEZq6F.png",
		},
		Color: 0xff0000,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Where?!",
				Value: whereMsg,
			},
			{
				Name:  "When?!",
				Value: whenMsg,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339), // Discord wants ISO8601; RFC3339 is an extension of ISO8601 and should be completely compatible.
		Title:     "Citadel running out of fuel, FEED IT!",
	}
}

func (b *fuelBot) loadStructures() ([]structureData, error) {
	v, err := b.tokenSource.Verify()
	if err != nil {
		return nil, errors.Wrap(err, "token verify error")
	}

	ctx := context.WithValue(context.Background(), goesi.ContextOAuth2, b.tokenSource)
	characterInfo, _, err := b.esi.ESI.CharacterApi.GetCharactersCharacterId(ctx, v.CharacterID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get character info")
	}

	corpStructures, _, err := b.esi.ESI.CorporationApi.GetCorporationsCorporationIdStructures(ctx, characterInfo.CorporationId, nil)
	if err != nil {
		e := err.(esi.GenericSwaggerError)
		return nil, errors.Wrapf(err, "unable to read corporation structures: %s", e.Model())
	}

	var out []structureData
	for _, structure := range corpStructures {
		structureInfo, _, err := b.esi.ESI.UniverseApi.GetUniverseStructuresStructureId(ctx, structure.StructureId, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to load strucutre info for structure: %d", structure.StructureId)
		}
		out = append(out, structureData{
			CorporationData: structure,
			UniverseData:    structureInfo,
		})
	}
	return out, nil
}

// shouldNotify checks if given strucutre should be notified
// right now.
func (b *fuelBot) shouldNotify(structure structureData) bool {
	expires := structure.CorporationData.FuelExpires
	// Structures already expired (unfueled).
	if expires.IsZero() {
		return false
	}
	if time.Until(expires) <= time.Duration(b.refuelNotification) {
		// If we already were notified, don't send message for notifyInterval duration.
		return !b.wasNotified(structure)
	}
	return false
}

// setWasNotified stores information that structure was already
// notified at time.Now()
func (b *fuelBot) setWasNotified(structure structureData) {
	id := structure.CorporationData.StructureId
	b.notified[id] = time.Now()
}

// wasNotified checks if this structure was notified within
// b.notifyInterval.
func (b *fuelBot) wasNotified(structure structureData) bool {
	id := structure.CorporationData.StructureId
	notifyTime, ok := b.notified[id]
	if !ok {
		return false
	}
	if time.Since(notifyTime) > b.notifyInterval {
		return false
	}
	return true
}
