package ui

import (
	"fmt"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/TheWeirdDev/Vodga/shared/utils"
	"github.com/TheWeirdDev/Vodga/ui/gtk_deprecated"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"io"
	"log"
	"net"
	"os/exec"
	"time"
)

type MainWindow struct {
	builder      *gtk.Builder
	window       *gtk.Window
	trayIcon     *gtk_deprecated.StatusIcon
	trayMenu     *gtk.Menu
	trayMenuItem *gtk.MenuItem
	server       net.Conn
	quit         chan struct{}
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
		connectToDaemon()
	}()
	mainWindow.quit = make(chan struct{})
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

	connectBtn, _ := (*utils.GetWidget(builder, "connect_btn")).(*gtk.Button)
	connectBtn.Connect("clicked", func() {
		mainWindow.server.Write([]byte("Hi\n"))
	})

	_, err2 := window.Connect("destroy", func() {
		close(mainWindow.quit)
		if mainWindow.server != nil {
			mainWindow.server.Close()
		}
		time.Sleep(20 * time.Millisecond)
		gtk.MainQuit()
	})

	if err2 != nil {
		log.Fatalf("Error: Cannot connect signals for GtkWindow")
	}
}

func listenToDaemon() {
	//scanner := bufio.NewScanner(mainWindow.server)
	buff := make([]byte, 100)

	for {
		_, err := mainWindow.server.Read(buff)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
			}
			break
		}
	}
	select {
	case <-mainWindow.quit:
		log.Println("Closed")

	default:
		log.Printf("Read error: \n")

		// You can't show dialogs in goroutines
		glib.IdleAdd(connectToDaemon)
	}

}

func connectToDaemon() {
	c, err := net.Dial("unix", consts.UnixSocket)
	if err != nil {

	firstDialog:
		msgDialog := gtk.MessageDialogNew(mainWindow.window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR,
			gtk.BUTTONS_YES_NO, "Vodga daemon is not running, do you want to start it?")
		response := msgDialog.Run()
		msgDialog.Destroy()
		if response == gtk.RESPONSE_YES {
			cmd := exec.Command("systemctl", "start", "vodga.service")
			if err := cmd.Start(); err != nil {
				log.Fatalf("cmd.Start: %v")
			}

			if err := cmd.Wait(); err != nil {
				if _, ok := err.(*exec.ExitError); ok {
					msgDialog2 := gtk.MessageDialogNew(mainWindow.window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR,
						gtk.BUTTONS_OK, "Cannot start the daemon")
					msgDialog2.Run()
					msgDialog2.Destroy()
					goto firstDialog
				} else {
					log.Fatalf("cmd.Wait: %v", err)
				}
			}
		} else {
			mainWindow.window.Close()
			return
		}
	}
	mainWindow.server = c
	go listenToDaemon()
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
