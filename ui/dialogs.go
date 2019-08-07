package ui

import (
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/gotk3/gotk3/gtk"
	"log"
)

func showImportSingleDialog(window* gtk.ApplicationWindow) {
	builder, err := gtk.BuilderNewFromFile(consts.AddSingelUI)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	dialog, _ := (*GetWidget(builder, "single_import_dialog")).(*gtk.Dialog)
	defer dialog.Destroy()
	dialog.SetParent(window)
	dialog.SetTransientFor(window)
	dialog.ShowAll()
	dialog.Run()
}