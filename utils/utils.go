package utils

import (
	"fmt"
	"github.com/TheWeirdDev/Vodga/utils/consts"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"log"
	"strings"
)


func GetWidget(builder *gtk.Builder, id string) *glib.IObject {
	widget, err := builder.GetObject(id)
	if err != nil {
		log.Fatalf("Error: Can't find widget: %q", id)
	}
	return &widget
}

func ErrorMsg(msg string) string {
	return fmt.Sprintf("%s %s", consts.MsgError, msg)
}

func OpenvpnEscape(unescaped string) string{
	escapedString := strings.ReplaceAll(unescaped, "\\", "\\\\")
	escapedString = strings.ReplaceAll(unescaped,"\"", "\\\"")
	escapedString = strings.ReplaceAll(unescaped,"\n", "\\n")

	if escapedString == unescaped && !strings.Contains(escapedString," ") &&
		!strings.Contains(escapedString,"#") && !strings.Contains(escapedString,";") &&
		!(escapedString == "") {
		return unescaped
	} else{
		return "\"" + escapedString + "\""
	}
}