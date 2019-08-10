package ui

import (
	"encoding/json"
	"github.com/TheWeirdDev/Vodga/shared/auth"
	"github.com/TheWeirdDev/Vodga/shared/utils"
	"io/ioutil"
	"os"
)

type cfg struct {
	Name       string `json:"name"`
	Creds      auth.Credentials `json:"creds"`
}

type singleCfg struct {
	cfg
	Port 	   uint `json:"port"`
	Proto      Proto `json:"proto"`
	Country    string `json:"country"`
	CountryISO string `json:"country_iso"`
}

type providerCfg struct {
	cfg
	Configs[] singleCfg `json:"configs"`
}

type data struct {
	Singles[] singleCfg `json:"single_configs"`
	Providers[] providerCfg `json:"providers"`
}

var dataPath = utils.UserHomeDir() + "/.config/vodga/vodga.json"

func checkDataDirectory() error {
	dir := utils.UserHomeDir() + "/.config/vodga/configs/"
	if _, err := os.Stat(dir); err == nil {
		return nil
	} else if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	} else {
		return err
	}
}

func getOrCreateData() (data, error) {
	empty := data{}
	if _, err := os.Stat(dataPath); err == nil {
		contents, err := ioutil.ReadFile(dataPath)
		if err != nil {
			return empty, nil
		}
		appData := data{}
		err = json.Unmarshal(contents, &appData)
		if err != nil {
			return empty, nil
		}
		return appData, nil
	} else if os.IsNotExist(err) {
		return empty, saveData(empty)
	} else {
		return empty, err
	}
}

func loadData() (data, error) {
	if err := checkDataDirectory(); err != nil {
		return data{}, err
	}
	return getOrCreateData()
}

func saveData(appData data) error {
	cfg, err := json.Marshal(appData)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dataPath, cfg, 0600)
	return err
}