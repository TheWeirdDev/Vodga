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
func getRemote(line string, db geoip2.Reader) (remote, error){
	rmt := remote{}
	// If you are using strings that may be invalid, check that ip is not nil
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return rmt, errors.New("Unknown remote option")
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
		ip =  net.ParseIP(fields[1])
	}
	rmt.ip = ip.String()

	record, err := db.Country(ip)
	if err != nil {
		return rmt, err
	}
	rmt.country = record.Country.Names["en"]
	rmt.countryIso = record.Country.IsoCode
	if len(fields) >= 2 {
		port, err := strconv.ParseUint(fields[2], 10, 32)
		if err != nil {
			return remote{}, err
		}
		rmt.port = uint(port)
	}

	if len(fields) >= 3 {
		rmt.proto = getProto(fields[3])
		if rmt.proto == "" {
			return remote{}, errors.New("Unknown protocol")
		}
	}
	return rmt, nil
}
func getConfig(file string, db geoip2.Reader) (config, error) {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	cfg := config{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(text, "remote") {
			rmt, err := getRemote(text, db)
			if err != nil {
				return config{}, err
			}
			cfg.remotes = append(cfg.remotes, rmt)
		}
		if strings.HasPrefix(text, "proto") {
			fields := strings.Fields(text)
			if len(fields) < 2 {
				return config{}, errors.New("Unknown proto option")
			}
			cfg.proto = getProto(fields[1])
			for _, rmt := range cfg.remotes {
				if rmt.proto == "" {
					rmt.proto = cfg.proto
				}
			}
		}
		//TODO: Needs more
	}

	if err := scanner.Err(); err != nil {
		return config{}, err
	}
	return cfg, nil
}
