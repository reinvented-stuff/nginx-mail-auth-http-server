package configuration

import (
	"encoding/json"
	"os"

	"github.com/rs/zerolog/log"
)

func ReadConfigurationFile(configPtr string, configuration *ConfigurationStruct) {

	log.Debug().Msgf("Loading configuration file '%s'", configPtr)

	configFile, _ := os.Open(configPtr)
	defer configFile.Close()

	JSONDecoder := json.NewDecoder(configFile)

	err := JSONDecoder.Decode(&configuration)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("stage", "init").
			Msgf("Error while loading configuration file '%s'", configPtr)
	}

	log.Debug().Msg("Finished loading configuration file")

}
