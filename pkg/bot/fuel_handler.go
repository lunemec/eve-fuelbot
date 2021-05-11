package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/antihax/goesi/esi"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

type structure struct {
	Name    string
	Effects []effect
}

type effect struct {
	Category   string
	Multiplier float64
}

var structureByType = map[int32]structure{
	35832: {
		Name: "Astrahus",
		Effects: []effect{
			{
				Category:   "citadel",
				Multiplier: 0.75,
			},
		},
	},
	35833: {
		Name: "Fortizar",
		Effects: []effect{
			{
				Category:   "citadel",
				Multiplier: 0.75,
			},
		},
	},
	35834: {
		Name: "Keepstar",
		Effects: []effect{
			{
				Category:   "citadel",
				Multiplier: 0.75,
			},
		},
	},
	35825: {
		Name: "Raitaru",
		Effects: []effect{
			{
				Category:   "engineering",
				Multiplier: 0.75,
			},
		},
	},
	35826: {
		Name: "Azbel",
		Effects: []effect{
			{
				Category:   "engineering",
				Multiplier: 0.75,
			},
		},
	},
	35827: {
		Name: "Sotiyo",
		Effects: []effect{
			{
				Category:   "engineering",
				Multiplier: 0.75,
			},
		},
	},
	35835: {
		Name: "Athanor",
		Effects: []effect{
			{
				Category:   "reaction",
				Multiplier: 0.8,
			},
			{
				Category:   "reprocessing",
				Multiplier: 0.8,
			},
		},
	},
	35836: {
		Name: "Tatara",
		Effects: []effect{
			{
				Category:   "reaction",
				Multiplier: 0.75,
			},
			{
				Category:   "reprocessing",
				Multiplier: 0.75,
			},
		},
	},
}

type service struct {
	Name        string
	FuelPerHour uint32
}

var serviceByCategory = map[string][]service{
	"citadel": {
		{
			Name:        "Clone Bay",
			FuelPerHour: 10,
		},
		{
			Name:        "Market",
			FuelPerHour: 40,
		},
	},
	"engineering": {
		{
			// All of these 3 services are actually 1 module with 1 fuel
			// fuel consumption.
			Name: "Blueprint Copying",
			// Name:        "Time Efficiency Research",
			// Name:        "Material Efficiency Research",
			FuelPerHour: 12,
		},
		{
			Name:        "Invention",
			FuelPerHour: 12, // Hyasoda module is 10.
		},
		{
			Name:        "Manufacturing (Standard)",
			FuelPerHour: 12,
		},
		{
			Name:        "Manufacturing (Capital)",
			FuelPerHour: 24,
		},
		{
			Name:        "Manufacturing (Supercapital)",
			FuelPerHour: 36,
		},
	},
	"reaction": {
		{
			Name:        "Biochemical Reactions",
			FuelPerHour: 15,
		},
		{
			Name:        "Composite Reactions",
			FuelPerHour: 15,
		},
		{
			Name:        "Hybrid Reactions",
			FuelPerHour: 15,
		},
	},
	"reprocessing": {
		{
			Name:        "Reprocessing",
			FuelPerHour: 10,
		},
	},
	"resource processing": {
		{
			Name:        "Moon mining", // TODO verify this name
			FuelPerHour: 5,
		},
	},
}

// messageFuelHandler will be called every time a new
// message is created on any channel that the autenticated bot has access to.
func (b *fuelBot) messageFuelHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// check if the message is "!fuel"
	if m.Content == "!fuel" {
		// Find the channel that the message came from.
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			b.log.Errorw("error finding channel_id to respond to !fuel message", "err", err)
			return
		}

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
	var (
		fields    []*discordgo.MessageEmbedField
		fuelTotal float64
	)

	for _, structureData := range structures {
		structureType := structureByTypeID(structureData.CorporationData.TypeId)
		// Add symbols for time ranges for fuel remaining. Green = OK
		symbol := ":green_square:"
		// < 7 days = orange
		if time.Until(structureData.CorporationData.FuelExpires) < 7*time.Hour*24 {
			symbol = ":orange_square:"
		}
		// < 1 day = red
		if time.Until(structureData.CorporationData.FuelExpires) < 1*time.Hour*24 {
			symbol = ":red_square:"
		}

		field := &discordgo.MessageEmbedField{
			Name: fmt.Sprintf("%s %s (%s)",
				symbol,
				structureData.UniverseData.Name,
				structureType.Name,
			),
		}
		fuelPerDay := b.structureFuelPerDay(structureData, structureType)
		fuelTotal += fuelPerDay

		if structureData.CorporationData.FuelExpires.IsZero() {
			field.Value = "`UNFUELLED`"
		} else {

			field.Value = fmt.Sprintf("`%s` (%s) \n **Services**: %s \n **Fuel per day**: %.0f",
				humanize.Time(structureData.CorporationData.FuelExpires),
				structureData.CorporationData.FuelExpires,
				formatServices(structureData.CorporationData.Services),
				fuelPerDay,
			)
		}
		fields = append(fields, field)
	}

	month := 30.0
	dailyFuelMsg := fmt.Sprintf("**Daily**: %.0f", fuelTotal)
	monthlyFuelMsg := fmt.Sprintf("**Monthly**: %.0f", fuelTotal*month)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fuelPrices, err := b.estFuelPrice(ctx)
	if err != nil {
		b.log.Errorw("unable to estimate fuel prices", "error", err)
	}
	fuelDailyPrices := formatFuelPrices(fuelTotal, fuelPrices)
	fuelMonthlyPrices := formatFuelPrices(fuelTotal*month, fuelPrices)

	finishedDailyMsg := fmt.Sprintf("%s %s", dailyFuelMsg, fuelDailyPrices)
	finishedMonthlyMsg := fmt.Sprintf("%s %s", monthlyFuelMsg, fuelMonthlyPrices)

	fields = append(fields, &discordgo.MessageEmbedField{
		Name: ":ice_cube: Total fuel",
		Value: fmt.Sprintf("%s \n%s",
			finishedDailyMsg,
			finishedMonthlyMsg,
		),
	})

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

func (b *fuelBot) structureFuelPerDay(structure structureData, structureType structure) float64 {
	var acc float64

	for _, service := range structure.CorporationData.Services {
		if service.State != "online" {
			continue
		}

		for serviceCategory, services := range serviceByCategory {
			for _, baseService := range services {
				if service.Name != baseService.Name {
					continue
				}

				var mul float64 = 1
				for _, effect := range structureType.Effects {
					if effect.Category == serviceCategory {
						mul = effect.Multiplier
					}
				}
				acc += float64(baseService.FuelPerHour) * mul
			}
		}
	}
	return acc * 24
}

func (b *fuelBot) estFuelPrice(ctx context.Context) (map[int32]float64, error) {
	var (
		regionID int32 = 10000002 // The Forge (Jita)
		typeIDs        = []int32{
			4247, // "Helium Fuel Block"
			4246, // "Hydrogen Fuel Block"
			4051, // "Nitrogen Fuel Block"
			4312, // "Oxygen Fuel Block"
		}
	)

	var out = make(map[int32]float64)
	for _, typeID := range typeIDs {
		priceHistory, _, err := b.esi.ESI.MarketApi.GetMarketsRegionIdHistory(ctx, regionID, typeID, nil)
		if err != nil {
			return nil, err
		}
		out[typeID] = movingAverage(7, priceHistory)
	}
	return out, nil
}

func movingAverage(days int, marketHistory []esi.GetMarketsRegionIdHistory200Ok) float64 {
	historyLen := len(marketHistory)
	if days > historyLen {
		days = historyLen
	}

	var accum float64
	for i := 1; i <= days; i++ {
		dayData := marketHistory[historyLen-i]
		accum += dayData.Average
	}

	return accum / float64(days)
}

func formatFuelPrices(blocks float64, fuelPrices map[int32]float64) string {
	if fuelPrices == nil {
		return "Error fetching prices."
	}

	// names must be indexed same as prices.
	names := []string{
		"[He]",
		"[H]",
		"[N]",
		"[O]",
	}
	prices := []float64{
		// Helium
		fuelPrices[4247] * blocks,
		// Hydrogen
		fuelPrices[4246] * blocks,
		// Nitrogen
		fuelPrices[4051] * blocks,
		// Oxygen
		fuelPrices[4312] * blocks,
	}
	var minPriceIdx int
	for i, price := range prices {
		if price < prices[minPriceIdx] {
			minPriceIdx = i
		}
	}

	var b strings.Builder
	b.WriteRune('\n')

	for i, price := range prices {
		// Lowest price is bold.
		if i == minPriceIdx {
			b.WriteString("**")
		}
		b.WriteString(names[i])
		b.WriteRune(' ')
		b.WriteString(humanize.CommafWithDigits(price, 0))
		// Lowest price is bold.
		if i == minPriceIdx {
			b.WriteString("**")
		}
		b.WriteString(" ISK")
		b.WriteRune('\n')
	}

	return b.String()
}

func structureByTypeID(typeID int32) structure {
	structureType, ok := structureByType[typeID]
	if !ok {
		return structure{
			Name: fmt.Sprintf("unknown structure type ID: %d", typeID),
		}
	}
	return structureType
}

func formatServices(services []esi.GetCorporationsCorporationIdStructuresService) string {
	var builder strings.Builder
	for i, service := range services {
		if i != 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(service.Name)
	}
	return builder.String()
}
