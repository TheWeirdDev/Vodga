package utils

import
(
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