package bot

import (
	"time"

	"github.com/antihax/goesi/esi"
)

// Testing structure data for example message.
var structures = []structureData{
	{
		CorporationData: esi.GetCorporationsCorporationIdStructures200Ok{
			TypeId:      35832,
			FuelExpires: time.Date(2021, 5, 11, 0, 0, 0, 0, time.UTC),
			Services: []esi.GetCorporationsCorporationIdStructuresService{
				{
					Name:  "Clone Bay",
					State: "online",
				},
			},
		},
		UniverseData: esi.GetUniverseStructuresStructureIdOk{
			Name: "Jita - My Astrahus < 1 day of fuel",
		},
	},
	{
		CorporationData: esi.GetCorporationsCorporationIdStructures200Ok{
			TypeId:      35835,
			FuelExpires: time.Date(2021, 5, 18, 0, 0, 0, 0, time.UTC),
			Services: []esi.GetCorporationsCorporationIdStructuresService{
				{
					Name:  "Reprocessing",
					State: "online",
				},
			},
		},
		UniverseData: esi.GetUniverseStructuresStructureIdOk{
			Name: "Jita - My Athanor < 7 days of fuel",
		},
	},
	{
		CorporationData: esi.GetCorporationsCorporationIdStructures200Ok{
			TypeId:      35825,
			FuelExpires: time.Date(2021, 10, 11, 0, 0, 0, 0, time.UTC),
			Services: []esi.GetCorporationsCorporationIdStructuresService{
				{
					Name:  "Manufacturing (Standard)",
					State: "online",
				},
			},
		},
		UniverseData: esi.GetUniverseStructuresStructureIdOk{
			Name: "Jita - My Raitaru > 7 days of fuel",
		},
	},
}
