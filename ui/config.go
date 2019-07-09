package ui

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/TheWeirdDev/Vodga/shared/auth"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/oschwald/geoip2-golang"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Proto string

const (
	udp Proto = "udp"
	tcp Proto = "tcp"
)

type remote struct {
	ip         string
	hostname   string
	country    string
	countryIso string
	port       uint
	proto      Proto
}

type config struct {
	path    string
	remotes []remote
	random  bool
	data    string
	proto   Proto
	creds   auth.Credentials
}

func getProto(p string) Proto {
	switch p {
	case "udp":
		return udp
	case "tcp":
		return tcp
	default:
		return ""
	}
}
func getRemote(line string, db *geoip2.Reader) (remote, error) {
	rmt := remote{}
	// If you are using strings that may be invalid, check that ip is not nil
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return rmt, errors.New("unknown remote option")
	}
	isIP, err := regexp.MatchString(consts.IPRegex, fields[1])
	if err != nil {
		return rmt, err
	}

	var ip net.IP
	if !isIP {
		rmt.hostname = fields[1]
		ips, err := net.LookupIP(fields[1])
		if err != nil {
			return rmt, err
		}
		ip = ips[0]
	} else {
		ip = net.ParseIP(fields[1])
	}
	rmt.ip = ip.String()

	record, err := db.Country(ip)
	if err != nil {
		return rmt, err
	}
	rmt.country = record.Country.Names["en"]
	rmt.countryIso = record.Country.IsoCode
	if len(fields) >= 3 {
		port, err := strconv.ParseUint(fields[2], 10, 32)
		if err != nil {
			return remote{}, err
		}
		rmt.port = uint(port)
	}

	if len(fields) >= 4 {
		rmt.proto = getProto(fields[3])
		if rmt.proto == "" {
			return remote{}, errors.New("unknown protocol")
		}
	}
	return rmt, nil
}

func readCredentials(line string, cfgPath string) (auth.Credentials, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return auth.Credentials{Auth: auth.USER_PASS}, nil
	}
	f, err := os.Open(fields[1])
	if err != nil {
		cfgPath += string(filepath.Separator)
		f2, err2 := os.Open(cfgPath + fields[1])
		if err2 != nil {
			return auth.Credentials{}, err
		}
		f = f2
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var creds []string

	for scanner.Scan() {
		if len(creds) >= 2 {
			break
		}
		text := scanner.Text()
		creds = append(creds, text)
	}
	if err := scanner.Err(); err != nil {
		return auth.Credentials{}, err
	}
	switch len(creds) {
	case 0:
		return auth.Credentials{Auth: auth.USER_PASS}, nil
	case 1:
		return auth.Credentials{Auth: auth.USER_PASS, Username: creds[0], Password: ""}, nil
	case 2:
		return auth.Credentials{Auth: auth.USER_PASS, Username: creds[0], Password: creds[1]}, nil
	default:
		return auth.Credentials{}, errors.New("unknown error while reading the credentials")
	}
}

func getConfig(file string, db *geoip2.Reader) (config, error) {
	f, err := os.Open(file)
	if err != nil {
		return config{}, err
	}
	defer f.Close()

	cfg := config{}
	cfg.data = ""
	cfg.creds.Auth = auth.NO_AUTH
	isClient := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if match, _ := regexp.MatchString("^remote\\s+.+$", text); match {
			rmt, err := getRemote(text, db)
			if err != nil {
				return config{}, err
			}
			cfg.remotes = append(cfg.remotes, rmt)
		} else if match, _ := regexp.MatchString("^proto\\s+.+$", text); match {
			fields := strings.Fields(text)
			if len(fields) < 2 {
				return config{}, errors.New("unknown proto option")
			}
			cfg.proto = getProto(fields[1])
		} else if text == "remote-random" {
			cfg.random = true
		} else if text == "client" {
			isClient = true
		} else if text == "auth-user-pass" {
			cfg.creds.Auth = auth.USER_PASS
			cfg.creds.Username = ""
			cfg.creds.Password = ""
		} else if match, _ := regexp.MatchString("^auth-user-pass\\s+.+$", text); match {
			dir, err := filepath.Abs(filepath.Dir(file))
			if err != nil {
				return config{}, err
			}
			if creds, err := readCredentials(text, dir); err != nil {
				return config{}, fmt.Errorf("unable to read the credentials: %v", err)
			} else {
				cfg.creds = creds
			}
		} else if match, _ := regexp.MatchString("^ca\\s+.+$", text); match {
			continue
		} else {
			if match, _ := regexp.MatchString("^[#;].*$", text); !match && text != "" {
				cfg.data += text + "\n"
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return config{}, err
	}

	if !isClient {
		return config{}, errors.New("not a client configuration (no 'client' option found)")
	}
	cfg.path = file
	if cfg.proto != "" {
		for i := range cfg.remotes {
			rmt := &cfg.remotes[i]
			if rmt.proto == "" {
				rmt.proto = cfg.proto
			}
		}
	} else {
		for _, rmt := range cfg.remotes {
			if rmt.proto != "" {
				cfg.proto = rmt.proto
				break
			}
		}
	}
	if len(cfg.remotes) == 0 || cfg.proto == "" {
		return config{}, errors.New("no remote or proto specified")
	}
	if len(cfg.remotes) == 1 && cfg.random {
		cfg.random = false
	}
	return cfg, nil
}
