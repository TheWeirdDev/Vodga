package ui

import (
	"encoding/json"
	"github.com/TheWeirdDev/Vodga/shared/auth"
	"github.com/TheWeirdDev/Vodga/shared/utils"
	"io/ioutil"
	"os"
)

type cfg struct {
	name       string
	creds      auth.Credentials
}

type singleCfg struct {
	cfg
	port 	   uint
	proto      Proto
	country    string
	countryISO string
}

type providerCfg struct {
	cfg
	configs[] singleCfg
}

type data struct {
	singles[] singleCfg
	providers[] providerCfg
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

func getOrCreateDataFile() (data, error) {
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
	return getOrCreateDataFile()
}

func saveData(appData data) error {
	cfg, err := json.Marshal(appData)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dataPath, cfg, 0600)
	return err
}