package chromeprofile

import (
	"chromium-websocket-proxy/config"
	"chromium-websocket-proxy/logger"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
)

const (
	ProfilesDir = "./profiles"
	ZipExt      = ".zip"
)

var tagProfileMap = make(map[string]string)

var once sync.Once

func LoadProfiles() {
	once.Do(func() {
		if !config.Get().GetChromeConfig().EnableCustomChromeProfiles {
			return
		}

		log := logger.Get()

		if _, err := os.Stat(ProfilesDir); os.IsNotExist(err) {
			err = os.Mkdir(ProfilesDir, os.ModePerm)
			if err != nil {
				log.Err(err).Msg("error initializing profiles directory")
			} else {
				log.Info().Msg("profiles directory initialized. No profiles to unzip")
			}
			return
		}
		files, err := os.ReadDir(ProfilesDir)
		if err != nil {
			log.Err(err).Msg("unable to unzip profiles")
			return
		}

		log.Info().Msg("attempting to unzip profiles")
		for _, file := range files {
			filePath := fmt.Sprintf("%s/%s", ProfilesDir, file.Name())
			ext := path.Ext(filePath)
			if ext == ZipExt {
				profile, err := unzipSource(filePath, ProfilesDir)
				profileName, _ := strings.CutSuffix(file.Name(), ZipExt)
				if err != nil {
					log.Err(err).Msg(fmt.Sprintf("unable to load %s profile", profileName))
				} else {
					tagProfileMap[profileName] = profile
					log.Info().Msg(fmt.Sprintf("loaded profile %s", profileName))
				}
			}
		}
		log.Info().Msg(fmt.Sprintf("unzipped %d profile(s)", len(tagProfileMap)))
	})
}

func GetProfileByTag(tag string) (string, bool) {
	profile, exists := tagProfileMap[tag]
	return profile, exists
}
