package ui

import (
	"fmt"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/oschwald/geoip2-golang"
	"regexp"
	"testing"
)

func TestGetConfig(t *testing.T){
	db, err := geoip2.Open(consts.GeoIPDataBase)
	if err != nil {
		t.Errorf("Test #1 failed: %v", err.Error())
	}
	defer db.Close()

	cfg,err := getConfig("config_test.ovpn", db)
	fmt.Printf("Got config: %v", cfg)
	if err != nil {
		t.Errorf("Test #1 failed: %v", err.Error())
	}
	if cfg.proto != udp {
		t.Errorf("Test #1 failed: proto mismatch")
	}
	if len(cfg.remotes) < 1 {
		t.Errorf("Test #1 failed: remote not found")
	} else {
		if cfg.remotes[0].port != 2744 ||  cfg.remotes[0].proto != udp {
			t.Errorf("Test #1 failed: remote port or proto mismatch")
		}
		if cfg.remotes[0].hostname != "freedome-at-gw.freedome-vpn.net"{
			t.Errorf("Test #1 failed: hostname mismatch")
		}
		match, _ := regexp.MatchString("188\\.172\\.220\\.(70|71|69)", cfg.remotes[0].ip)
		if !match {
			t.Errorf("Test #1 failed: ip mismatch")
		}
	}
}
