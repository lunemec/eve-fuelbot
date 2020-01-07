package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

// messageFuelHandler will be called every time a new
// message is created on any channel that the autenticated bot has access to.
func (b *fuelBot) messageFuelHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// check if the message is "!airhorn"
	if m.Content == "!fuel" {
		// Find the channel that the message came from.
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			b.log.Errorw("error finding channel_id to respond to !fuel message", "err", err)
			return
		}
		fmt.Printf("CHANNEL_ID: %s", c.ID)

		structs, err := b.loadStructures()
		if err != nil {
			b.log.Errorw("error loading structure information", "err", err)
			return
		}
		b.log.Infow("Sending response to !fuel command",
			"channel_id", b.channelID,
		)
		_, err = b.discord.ChannelMessageSendEmbed(c.ID, b.allStructuresMessage(structs))
		if err != nil {
			b.log.Errorw("error sending discord message", "err", err)
			return
		}
	}
}

func (b *fuelBot) allStructuresMessage(structures []structureData) *discordgo.MessageEmbed {
	var fields []*discordgo.MessageEmbedField
	for _, structure := range structures {
		field := &discordgo.MessageEmbedField{
			Name: structure.UniverseData.Name,
		}
		if structure.CorporationData.FuelExpires.IsZero() {
			field.Value = "`UNFUELLED`"
		} else {
			field.Value = fmt.Sprintf("`%s` (%s)",
				humanize.Time(structure.CorporationData.FuelExpires),
				structure.CorporationData.FuelExpires,
			)
		}
		fields = append(fields, field)
	}

	return &discordgo.MessageEmbed{
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://i.imgur.com/pKEZq6F.png",
		},
		Color:     0x00ff00,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339), // Discord wants ISO8601; RFC3339 is an extension of ISO8601 and should be completely compatible.
		Title:     "Feeding status",
	}
}
