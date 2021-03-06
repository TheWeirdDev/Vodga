package ui

import (
	"github.com/TheWeirdDev/Vodga/shared/auth"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/gotk3/gotk3/gtk"
	"log"
	"strconv"
)

func (gui *mainGUI) showImportSingleDialog() {
	builder, err := gtk.BuilderNewFromFile(consts.AddSingelUI)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	var cfg config
	selected := false

	errorBar, _ := (*GetWidget(builder, "bar_error")).(*gtk.InfoBar)
	errorLabel, _ := (*GetWidget(builder, "lbl_error")).(*gtk.Label)

	dialog, _ := (*GetWidget(builder, "single_import_dialog")).(*gtk.Dialog)
	cancelBtn, _ := (*GetWidget(builder, "btn_cancel")).(*gtk.Button)
	_, _ = cancelBtn.Connect("clicked", func() {
		dialog.Close()
	})

	importBtn, _ := (*GetWidget(builder, "btn_import")).(*gtk.Button)
	_, _ = importBtn.Connect("clicked", func() {
		if selected {
			dialog.Close()
		} else {
			errorBar.SetProperty("revealed", true)
			errorLabel.SetText("No config file is selected")
		}
	})

	pathEntry, _ := (*GetWidget(builder, "entry_path")).(*gtk.Entry)
	errorBar.Connect("response", func() {
		errorBar.SetProperty("revealed", false)
	})

	detailsGrid, _ := (*GetWidget(builder, "grid_details")).(*gtk.Grid)
	detailsGrid.SetVisible(false)

	remoteLabel, _ := (*GetWidget(builder, "lbl_remote")).(*gtk.Label)
	countryLabel, _ := (*GetWidget(builder, "lbl_country")).(*gtk.Label)
	protoLabel, _ := (*GetWidget(builder, "lbl_proto")).(*gtk.Label)
	authCheckbox, _ := (*GetWidget(builder, "chk_password")).(*gtk.CheckButton)
	authBox, _ := (*GetWidget(builder, "box_auth")).(*gtk.Box)
	userEntrry, _ := (*GetWidget(builder, "entry_username")).(*gtk.Entry)
	passEntry, _ := (*GetWidget(builder, "entry_password")).(*gtk.Entry)

	browseBtn, _ := (*GetWidget(builder, "btn_browse")).(*gtk.Button)
	_, _ = browseBtn.Connect("clicked", func() {
		errorBar.SetProperty("revealed", false)

		fileChooser, err := gtk.FileChooserDialogNewWith2Buttons("Choose openvpn config file", dialog,
		gtk.FILE_CHOOSER_ACTION_OPEN, "Open", gtk.RESPONSE_ACCEPT,
		"Cancel", gtk.RESPONSE_CANCEL)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		filter, err := gtk.FileFilterNew()
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		filter.SetName("OpenVPN Configuration files")
		filter.AddPattern("*.ovpn")
		filter.AddPattern("*.conf")
		fileChooser.SetFilter(filter)
		defer fileChooser.Destroy()
		fileChooser.ShowAll()
		response := fileChooser.Run()
		if response != gtk.RESPONSE_ACCEPT {
			return
		}

		filePath := fileChooser.GetFilename()
		cfg, err = getConfig(filePath, true)
		if err != nil {
			errorBar.SetProperty("revealed", true)
			pathEntry.SetText("")
			errorLabel.SetText("Error: " + err.Error())
			selected = false
			detailsGrid.Hide()
			return
		}

		selected = true
		remoteLabel.SetText(cfg.remotes[0].ips[0] + ":" + strconv.FormatUint(uint64(cfg.remotes[0].port), 10))
		countryLabel.SetText(cfg.remotes[0].countryIso + ", " + cfg.remotes[0].country)
		protoLabel.SetText(string(cfg.proto))
		authCheckbox.Connect("toggled", func() {
			authBox.SetVisible(authCheckbox.GetActive())
		})
		showAuth := cfg.creds.Auth == auth.USER_PASS
		authCheckbox.SetActive(showAuth)
		authBox.SetVisible(showAuth)
		if showAuth {
			userEntrry.SetText(cfg.creds.Username)
			passEntry.SetText(cfg.creds.Password)
		}
		
		detailsGrid.SetVisible(true)
		detailsGrid.ShowAll()
		pathEntry.SetText(filePath)
	})

	defer dialog.Destroy()
	dialog.SetTransientFor(gui.window)
	dialog.ShowAll()
	dialog.Run()
}