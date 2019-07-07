package ui

import (
	"bufio"
	"errors"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/oschwald/geoip2-golang"
	"log"
	"net"
	"os"
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
func getConfig(file string, db *geoip2.Reader) (config, error) {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	cfg := config{}
	cfg.data = ""
	isClient := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		cfg.data += text + "\n"
		if match, _ := regexp.MatchString("^remote\\s+", text); match {
			rmt, err := getRemote(text, db)
			if err != nil {
				return config{}, err
			}
			cfg.remotes = append(cfg.remotes, rmt)
		} else if match, _ := regexp.MatchString("^proto\\s+", text); match {
			fields := strings.Fields(text)
			if len(fields) < 2 {
				return config{}, errors.New("unknown proto option")
			}
			cfg.proto = getProto(fields[1])
		} else if text == "remote-random" {
			cfg.random = true
		} else if text == "client" {
			isClient = true
		}
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
	if err := scanner.Err(); err != nil {
		return config{}, err
	}
	return cfg, nil
}
