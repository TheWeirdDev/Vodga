package ui

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/TheWeirdDev/Vodga/shared/auth"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/oschwald/geoip2-golang"
	"io/ioutil"
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
	ips        []string
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
	proto   Proto
	creds   auth.Credentials
	ca      string
	cert    string
	key     string
	tlsAuth string
	other   string
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

// Parses a 'remote' option into a struct
func getRemote(line string, db *geoip2.Reader) (remote, error) {
	rmt := remote{}

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return rmt, errors.New("unknown remote option")
	}
	isIP, err := regexp.MatchString(consts.IPRegex, fields[1])
	if err != nil {
		return rmt, err
	}

	var ips []net.IP
	// Lookup ip address if remote is not an IP
	if !isIP {
		rmt.hostname = fields[1]
		ip4, err := net.LookupIP(fields[1])
		if err != nil {
			return rmt, err
		}

		for _, ip := range ip4 {
			if ip.To4() != nil {
				ips = append(ips, ip)
			}
		}
	} else {
		ips = append(ips, net.ParseIP(fields[1]))
	}
	if len(ips) == 0 {
		return remote{}, errors.New("can't resolve domain name")
	}

	for _, ip := range ips {
		rmt.ips = append(rmt.ips, ip.String())
	}

	// Fetch country info from remotes
	// Only one is needed because we assume all of them are from the same country
	var record *geoip2.Country
	var dberr error
	for _, ip := range ips {
		record, dberr = db.Country(ip)
		if dberr == nil {
			rmt.country = record.Country.Names["en"]
			rmt.countryIso = record.Country.IsoCode
			break
		}
	}
	if dberr != nil {
		return remote{}, dberr
	}
	// TODO: Check for empty country field while importing

	// port is provided in remote option
	if len(fields) >= 3 {
		port, err := strconv.ParseUint(fields[2], 10, 32)
		if err != nil {
			return remote{}, err
		}
		rmt.port = uint(port)
	}

	// proto is provided in remote option
	if len(fields) >= 4 {
		rmt.proto = getProto(fields[3])
		if rmt.proto == "" {
			return remote{}, errors.New("unknown protocol")
		}
	}
	return rmt, nil
}

// Read credentials from an external text file
func readCredentials(line string, cfgPath string) (auth.Credentials, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return auth.Credentials{Auth: auth.USER_PASS}, nil
	}
	f, err := os.Open(fields[1])
	// If the path doesn't exist, check the relative path
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

	// The first line is the username (mandatory)
	// The second line is the password (optional)
	// The rest of the file will be ignored just like openvpn does
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

// Read external certificates
func readCert(line string, cfgPath string) (string, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", nil
	}
	f, err := os.Open(fields[1])
	// check the relative path
	if err != nil {
		cfgPath += string(filepath.Separator)
		f2, err2 := os.Open(cfgPath + fields[1])
		if err2 != nil {
			return "", err
		}
		f = f2
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// getConfig reads the configuration file and gather all the info needed
// and parses it into structures that we can store
func getConfig(file string, db *geoip2.Reader, single bool) (config, error) {
	f, err := os.Open(file)
	if err != nil {
		return config{}, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return config{}, err
	}
	// The config file size shouldn't be more than "100 KB"
	if stat.Size() > 100*1024 {
		return config{}, errors.New("the file is too big")
	}

	dir, err := filepath.Abs(filepath.Dir(file))
	if err != nil {
		return config{}, err
	}

	cfg := config{}
	cfg.other = ""
	cfg.creds.Auth = auth.NO_AUTH

	isClient := false

	// Keep track of reading inline certificates
	isReadingCa := false
	isReadingCert := false
	isReadingKey := false
	isReadingTLSAuth := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if isReadingCa {
			if text == "</ca>" {
				isReadingCa = false
			} else {
				cfg.ca += text + "\n"
			}
			continue
		} else if isReadingCert {
			if text == "</cert>" {
				isReadingCert = false
			} else {
				cfg.cert += text + "\n"
			}
			continue
		} else if isReadingKey {
			if text == "</key>" {
				isReadingKey = false
			} else {
				cfg.key += text + "\n"
			}
			continue
		} else if isReadingTLSAuth {
			if text == "</tls-auth>" {
				isReadingTLSAuth = false
			} else {
				cfg.tlsAuth += text + "\n"
			}
			continue
		}

		// Parse every option we need and save the rest in cfg.other
		if match, _ := regexp.MatchString("^remote\\s+.+$", text); match {
			rmt, err := getRemote(text, db)
			if err != nil {
				return config{}, err
			}
			cfg.remotes = append(cfg.remotes, rmt)
			if len(rmt.ips) > 1 {
				cfg.random = true
			}
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
		} else if text == "auth-user-pass" && single{
			cfg.creds.Auth = auth.USER_PASS
			cfg.creds.Username = ""
			cfg.creds.Password = ""
		} else if match, _ := regexp.MatchString("^auth-user-pass\\s+.+$", text);
			match && single {
			if creds, err := readCredentials(text, dir); err != nil {
				return config{}, fmt.Errorf("unable to read the credentials: %v", err)
			} else {
				cfg.creds = creds
			}
		} else if match, _ := regexp.MatchString("^ca\\s+.+$", text); match {
			if ca, err := readCert(text, dir); err != nil {
				return config{}, fmt.Errorf("unable to read the ca file: %v", err)
			} else {
				cfg.ca = ca
			}
		} else if match, _ := regexp.MatchString("^cert\\s+.+$", text); match {
			if cert, err := readCert(text, dir); err != nil {
				return config{}, fmt.Errorf("unable to read the cert file: %v", err)
			} else {
				cfg.cert = cert
			}
		} else if match, _ := regexp.MatchString("^key\\s+.+$", text); match {
			if key, err := readCert(text, dir); err != nil {
				return config{}, fmt.Errorf("unable to read the key file: %v", err)
			} else {
				cfg.key = key
			}
		} else if match, _ := regexp.MatchString("^tls-auth\\s+.+$", text); match {
			if tlsAuth, err := readCert(text, dir); err != nil {
				return config{}, fmt.Errorf("unable to read the tls-auth file: %v", err)
			} else {
				cfg.tlsAuth = tlsAuth
			}
		} else if text == "<ca>" {
			isReadingCa = true
		} else if text == "<cert>" {
			isReadingCert = true
		} else if text == "<key>" {
			isReadingKey = true
		} else if text == "<tls-auth>" {
			isReadingTLSAuth = true
		} else {
			// Comments and management options will be deleted
			comment, _ := regexp.MatchString("^[#;].*$", text)
			mgmt, _ := regexp.MatchString("^management.*$", text)
			if !comment && !mgmt && text != "" {
				cfg.other += text + "\n"
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return config{}, err
	}
	if isReadingCa || isReadingCert || isReadingKey || isReadingTLSAuth {
		return config{}, errors.New("config file is corrupted")
	}
	if !isClient {
		return config{}, errors.New("not a client configuration (no 'client' option found)")
	}
	cfg.path = file

 checkProto:
	if cfg.proto != "" {
		// If config has a proto and some of it remotes don't, add it to them
		for i := range cfg.remotes {
			rmt := &cfg.remotes[i]
			if rmt.proto == "" {
				rmt.proto = cfg.proto
			}
		}
	} else {
		// If config don't have a proto, get it from the first remote that has
		for _, rmt := range cfg.remotes {
			if rmt.proto != "" {
				cfg.proto = rmt.proto
				// Now add this proto to other remotes that don't have proto
				goto checkProto
			}
		}
	}
	if cfg.ca == "" {
		return config{}, errors.New("no 'ca' option specified")
	}
	if (cfg.cert == "" && cfg.key != "") || (cfg.cert != "" && cfg.key == "") {
		return config{}, errors.New("'cert' and 'key' options must be used together")
	}
	if len(cfg.remotes) == 0 || cfg.proto == "" {
		return config{}, errors.New("no 'remote' or 'proto' option specified")
	}
	return cfg, nil
}
