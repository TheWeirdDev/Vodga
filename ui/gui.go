package ui

import (
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/TheWeirdDev/Vodga/shared/utils"
	"github.com/TheWeirdDev/Vodga/ui/gtk_deprecated"
	"github.com/gotk3/gotk3/gtk"
	"log"
)

type MainWindow struct {
	builder      *gtk.Builder
	window       *gtk.Window
	trayIcon     *gtk_deprecated.StatusIcon
	trayMenu     *gtk.Menu
	trayMenuItem *gtk.MenuItem
}

var mainWindow = &MainWindow{}
var initDone = false

func StartGui() {
	if initDone {
		log.Fatalf("Error: GUI is already Initialized")
		return
	}
	defer func() {
		initDone = true
		initWidgets()
		showMainWindow()
	}()

	builder, err := gtk.BuilderNewFromFile(consts.UIFilePath)
	if err != nil {
		log.Fatalf("Error: Can not initialize the ui builer")
	}
	mainWindow.builder = builder

	window, ok := (*utils.GetWidget(builder, "main_window")).(*gtk.Window)
	if !ok {
		log.Fatalf("Error: GtkWindow not found")
	}
	mainWindow.window = window

	_, err2 := window.Connect("destroy", func() {
		gtk.MainQuit()
	})
	if err2 != nil {
		log.Fatalf("Error: Cannot connect signals for GtkWindow")
	}
}

func initWidgets() {
	statusIcon, err := gtk_deprecated.StatusIconNewFromIconName("gtk-disconnect")
	if err != nil {
		log.Fatalf("Error: Cannot create tray icon")
	}
	mainWindow.trayIcon = statusIcon

	menu, err := gtk.MenuNew()
	if err != nil {
		log.Fatalf("Error: Cannot create tray menu")
	}
	defer menu.ShowAll()

	mainWindow.trayMenu = menu

	menuItemExit, err := gtk.MenuItemNewWithLabel("Exit")
	if err != nil {
		log.Fatalf("Error: Cannot create tray menu")
	}

	if _, err = menuItemExit.Connect("activate", func() {
		gtk.MainQuit()
	}); err != nil {
		log.Fatalf("Error: Cannot connect menu item")
	}

	mainWindow.trayMenuItem = menuItemExit
	mainWindow.trayMenu.Append(mainWindow.trayMenuItem)
	if _, err = mainWindow.trayIcon.Connect("activate", func() {
		if !mainWindow.window.IsVisible() {
			mainWindow.window.SetVisible(true)
		} else if !mainWindow.window.IsActive() {
			mainWindow.window.Present()
		} else {
			mainWindow.window.SetVisible(false)
		}
	}); err != nil {
		log.Fatalf("Error: Cannot connect menu item")
	}

	_, err = mainWindow.trayIcon.Connect("popup_menu",
		func(icon interface{}, a uint, b uint32) {
			mainWindow.trayIcon.PopupMenu(mainWindow.trayMenu, a, b)
		})

	if err != nil {
		log.Fatalf("Error: Cannot connect tray popup menu")
	}
}

func showMainWindow() {
	if mainWindow.window == nil {
		log.Fatalf("Error: Main window is not initialized")
	}
	mainWindow.window.ShowAll()
}
