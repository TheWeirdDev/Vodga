package ui

import (
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/gotk3/gotk3/gtk"
	"github.com/oschwald/geoip2-golang"
	"log"
)

func (gui *mainGUI) showImportSingleDialog() {
	builder, err := gtk.BuilderNewFromFile(consts.AddSingelUI)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	dialog, _ := (*GetWidget(builder, "single_import_dialog")).(*gtk.Dialog)
	cancelBtn, _ := (*GetWidget(builder, "btn_cancel")).(*gtk.Button)
	_, _ = cancelBtn.Connect("clicked", func() {
		dialog.Close()
	})

	importBtn, _ := (*GetWidget(builder, "btn_import")).(*gtk.Button)
	_, _ = importBtn.Connect("clicked", func() {

		dialog.Close()
	})

	pathEntry, _ := (*GetWidget(builder, "entry_path")).(*gtk.Entry)
	errorBar, _ := (*GetWidget(builder, "bar_error")).(*gtk.InfoBar)
	errorLabel, _ := (*GetWidget(builder, "lbl_error")).(*gtk.Label)
	errorBar.Connect("response", func() {
		errorBar.SetProperty("revealed", false)
	})

	browseBtn, _ := (*GetWidget(builder, "btn_browse")).(*gtk.Button)
	_, _ = browseBtn.Connect("clicked", func() {
		errorBar.SetProperty("revealed", false)

		fileChooser, err := gtk.FileChooserDialogNewWith2Buttons("Choose openvpn config file", dialog,
		gtk.FILE_CHOOSER_ACTION_OPEN, "Open", gtk.RESPONSE_ACCEPT,
		"Cancel", gtk.RESPONSE_CANCEL)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		defer fileChooser.Destroy()
		fileChooser.ShowAll()
		response := fileChooser.Run()
		if response != gtk.RESPONSE_ACCEPT {
			return
		}

		filePath := fileChooser.GetFilename()
		db, err := geoip2.Open(consts.GeoIPDataBase)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		defer db.Close()
		_, err = getConfig(filePath, db, true)
		if err != nil {
			errorBar.SetProperty("revealed", true)
			pathEntry.SetText("")
			errorLabel.SetText("Error: " + err.Error())
			return
		}
		pathEntry.SetText(filePath)
	})

	defer dialog.Destroy()
	dialog.SetTransientFor(gui.window)
	dialog.ShowAll()
	dialog.Run()
}