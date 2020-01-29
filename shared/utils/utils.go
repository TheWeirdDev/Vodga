package utils

import (
	"errors"
	"log"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

func BytecountToString(in, out, tin, tout uint64) (string, string, string, string) {
	return strconv.FormatUint(in, 10),
		strconv.FormatUint(out, 10),
		strconv.FormatUint(tin, 10),
		strconv.FormatUint(tout, 10)
}

func BytecountToUint(in, out, tin, tout string)(uint64, uint64, uint64, uint64){
	i ,_ := strconv.ParseUint(in, 10, 64)
	o ,_ := strconv.ParseUint(out, 10, 64)
	ti ,_ := strconv.ParseUint(tin, 10, 64)
	to ,_ := strconv.ParseUint(tout, 10, 64)
	return i, o, ti, to
}

func OpenvpnEscape(unescaped string) string {
	escapedString := strings.ReplaceAll(unescaped, "\\", "\\\\")
	escapedString = strings.ReplaceAll(escapedString, "\"", "\\\"")
	escapedString = strings.ReplaceAll(escapedString, "\n", "\\n")

	if escapedString == unescaped && !strings.Contains(escapedString, " ") &&
		!strings.Contains(escapedString, "#") && !strings.Contains(escapedString, ";") &&
		!(escapedString == "") {
		return unescaped
	} else {
		return "\"" + escapedString + "\""
	}
}

func GetGeoIPData(ip string) (string, string, error) {
	o, err := exec.Command("geoiplookup", ip).Output()
	if err != nil {
		return "", "", err
	}
	text, out := "GeoIP Country Edition: ", string(o)
	if strings.Contains(out, text) {
		start := strings.Index(out, text) + len(text)
		end := start + strings.Index(out[start:], "\n")
		data := strings.FieldsFunc(out[start:end], func(s rune) bool {return s == ','})
		if len(data) < 2 {
			return "", "", errors.New(data[0])
		}
		return strings.TrimSpace(data[0]), strings.TrimSpace(data[1]), nil
	}
	return "", "", err
}

func UserHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}
