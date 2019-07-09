package ui

import (
	"github.com/TheWeirdDev/Vodga/shared/auth"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/oschwald/geoip2-golang"
	"regexp"
	"strings"
	"testing"
)

func TestGetConfig(t *testing.T) {
	db, err := geoip2.Open(consts.GeoIPDataBase)
	if err != nil {
		t.Errorf("Test #1 failed: %v", err.Error())
	}
	defer db.Close()

	cfg, err := getConfig("data/test/config_test.ovpn", db)

	if err != nil {
		t.Errorf("Test #1 failed: %v", err.Error())
	}
	if cfg.proto != udp {
		t.Errorf("Test #1 failed: proto mismatch")
	}
	if len(cfg.remotes) < 1 {
		t.Errorf("Test #1 failed: remote not found")
	} else {
		if cfg.remotes[0].port != 2744 || cfg.remotes[0].proto != udp {
			t.Errorf("Test #1 failed: remote port or proto mismatch")
		}
		if cfg.remotes[0].hostname != "freedome-at-gw.freedome-vpn.net" {
			t.Errorf("Test #1 failed: hostname mismatch")
		}
		match, _ := regexp.MatchString("^188\\.172\\.220\\.(70|71|69)$", cfg.remotes[0].ip)
		if !match {
			t.Errorf("Test #1 failed: ip mismatch")
		}
	}
	if cfg.random {
		t.Errorf("Test #1 failed: random should be false")
	}
	if cfg.creds.Auth != auth.NO_AUTH {
		t.Errorf("Test #1 failed: auth method is wrong")
	}
}

func TestGetConfigWithCredentials(t *testing.T) {
	db, err := geoip2.Open(consts.GeoIPDataBase)
	if err != nil {
		t.Errorf("Test #2 failed: %v", err.Error())
	}
	defer db.Close()

	cfg, err := getConfig("data/test/config_test2.ovpn", db)

	if err != nil {
		t.Errorf("Test #2 failed: %v", err.Error())
	}

	if cfg.creds.Auth != auth.USER_PASS {
		t.Errorf("Test #2 failed: auth method is wrong")
	}
	if cfg.creds.Username != "test_username" || cfg.creds.Password != "test_password" {
		t.Errorf("Test #2 failed: wrong credentials")
	}
	if !strings.Contains(cfg.ca, "TESTTESTTEST") {
		t.Errorf("Test #2 failed: wrong ca")
	}
	if !strings.Contains(cfg.cert, "TESTCERTTESTCERT") {
		t.Errorf("Test #2 failed: wrong cert")
	}
	if !strings.Contains(cfg.key, "TESTKEYTESTKEY") {
		t.Errorf("Test #2 failed: wrong key")
	}
}
