package bot

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"eve-fuelbot/pkg/token"

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
	log.Infow("EVE FuelBot starting", "check_interval", checkInterval, "notify_interval", notifyInterval, "refuel_notification", refuelNotification)
	esi := goesi.NewAPIClient(client, "EVE FuelBot")
	return &fuelBot{
		tokenSource:        tokenSource,
		log:                log,
		esi:                esi,
		discord:            discord,
		channelID:          channelID,
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

	for {
		structs, err := b.loadStructures()
		if err != nil {
			return errors.Wrap(err, "error loading structure information")
		}

		for _, structure := range structs {
			notify := b.shouldNotify(structure)
			if notify {
				b.log.Infow("Sending message",
					"channel_id", b.channelID,
					"structure_id", structure.CorporationData.StructureId,
					"structure_name", structure.UniverseData.Name,
				)
				_, err = b.discord.ChannelMessageSendEmbed(b.channelID, b.message(&structure))
				if err != nil {
					return errors.Wrap(err, "error sending discord message")
				}
			}
		}
		time.Sleep(b.checkInterval)
	}
}

func (b *fuelBot) message(structure *structureData) *discordgo.MessageEmbed {
	whereMsg := "`%s`"
	whereMsg = fmt.Sprintf(whereMsg, structure.UniverseData.Name)

	whenMsg := "`%s` (%s)"
	whenMsg = fmt.Sprintf("`%s` (%s)", humanize.Time(structure.CorporationData.FuelExpires), structure.CorporationData.FuelExpires)

	return &discordgo.MessageEmbed{
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://i.imgur.com/pKEZq6F.png",
		},
		Color: 0xff0000,
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:  "Where?!",
				Value: whereMsg,
			},
			&discordgo.MessageEmbedField{
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

func (b *fuelBot) shouldNotify(structure structureData) bool {
	expires := structure.CorporationData.FuelExpires
	if time.Until(expires) <= time.Duration(b.refuelNotification) {
		// If we already were notified, don't send message for notifyInterval duration.
		if b.wasNotified(structure) {
			return false
		}
		return true
	}
	return false
}

func (b *fuelBot) wasNotified(structure structureData) bool {
	id := structure.CorporationData.StructureId
	notifyTime, ok := b.notified[id]
	if !ok {
		b.notified[id] = time.Now()
		return false
	}
	if time.Since(notifyTime) > b.notifyInterval {
		b.notified[id] = time.Now()
		return false
	}
	return true
}

func (b *fuelBot) formatMessage(structure structureData) string {
	msg := `**%s** is running out of fuel!
Will run out in **%s** (%s).`
	return fmt.Sprintf(msg,
		structure.UniverseData.Name,
		humanize.Time(structure.CorporationData.FuelExpires),
		structure.CorporationData.FuelExpires,
	)
}
