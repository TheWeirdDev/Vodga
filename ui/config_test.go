package ui

import (
	"github.com/TheWeirdDev/Vodga/shared/auth"
	"regexp"
	"strings"
	"testing"
)

func TestGetConfig(t *testing.T) {
	cfg, err := getConfig("data/test/config_test.ovpn", true)

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
		if len(cfg.remotes[0].ips) != 3 {
			t.Errorf("Test #1 Failed: invalid ips")
		}

		for _,ip := range cfg.remotes[0].ips {
			match, _ := regexp.MatchString("^188\\.172\\.220\\.(70|71|69)$", ip)
			if !match {
				t.Errorf("Test #1 failed: ip mismatch")
			}
		}
	}
	if !cfg.random {
		t.Errorf("Test #1 failed: random should be true")
	}
	if cfg.creds.Auth != auth.NO_AUTH {
		t.Errorf("Test #1 failed: auth method is wrong")
	}
	//fmt.Printf("%v\n", cfg.remotes)
}

func TestGetConfigWithCredentials(t *testing.T) {

	cfg, err := getConfig("data/test/config_test2.ovpn", true)

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
	if cfg.tlsAuth == "" {
		t.Errorf("Test #2 failed: wrong tls auth")
	}
	//fmt.Printf("%v\n", cfg.remotes)
}
