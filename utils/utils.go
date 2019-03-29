package utils

import
(
	"fmt"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"log"
)


func GetWidget(builder *gtk.Builder, id string) *glib.IObject {
	widget, err := builder.GetObject(id)
	if err != nil {
		log.Fatalf("Error: Can't find widget: %q", id)
	}
	return &widget
}

func EnsureEnoughArguments(args []string, count int) error {
	c := len(args) - 1
	if c != count {
		return fmt.Errorf("command %q takes %d argument(s) but %d were given", args[0], count, c)
	}
	return nil
}