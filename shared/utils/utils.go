package utils

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"log"
	"os/user"
	"strconv"
	"strings"
)

func GetWidget(builder *gtk.Builder, id string) *glib.IObject {
	widget, err := builder.GetObject(id)
	if err != nil {
		log.Fatalf("Error: Can't find widget: %q", id)
	}
	return &widget
}

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

func UserHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}
