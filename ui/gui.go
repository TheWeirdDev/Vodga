package ui

import (
	"bufio"
	"fmt"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/TheWeirdDev/Vodga/shared/messages"
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

type mainGUI struct {
	builder      *gtk.Builder
	window       *gtk.Window
	trayIcon     *gtk_deprecated.StatusIcon
	trayMenu     *gtk.Menu
	trayMenuItem *gtk.MenuItem
	server       net.Conn
	state        string
	quit         chan struct{}
}

func CreateGUI() *mainGUI {
	maingui := &mainGUI{}
	return maingui
}

func (gui *mainGUI) Run() {
	defer func() {
		gui.initWidgets()
		gui.showMainWindow()
		gui.connectToDaemon()
	}()

	gui.quit = make(chan struct{})
	builder, err := gtk.BuilderNewFromFile(consts.UIFilePath)
	if err != nil {
		log.Fatalf("Error: Can not initialize the ui builer")
	}
	gui.builder = builder

	window, ok := (*utils.GetWidget(builder, "main_window")).(*gtk.Window)
	if !ok {
		log.Fatalf("Error: GtkWindow not found")
	}
	gui.window = window

	connectBtn, _ := (*utils.GetWidget(builder, "connect_btn")).(*gtk.Button)
	_, _ = connectBtn.Connect("clicked", func() {

	})

	addCfgBtn , _ := (*utils.GetWidget(builder, "btn_add_config")).(*gtk.Button)
	_, _ = addCfgBtn.Connect("clicked", func() {

	})
	_, _ = window.Connect("destroy", func() {
		close(gui.quit)
		if gui.server != nil {
			gui.server.Close()
		}
		time.Sleep(20 * time.Millisecond)
		gtk.MainQuit()
	})

	go func() {
		// Check for bandwidth usage every 1 second
		tck := time.Tick(time.Second)
		for range tck {
			if gui.server != nil && gui.state == consts.StateCONNECTED {
				messages.SendMessage(messages.GetBytecountMsg(), gui.server)
			}
		}
	}()
}

func (gui *mainGUI) listenToDaemon() {
	scanner := bufio.NewScanner(gui.server)

	for scanner.Scan() {
		text := scanner.Text()
		msg, err := messages.UnmarshalMsg(text)
		if err != nil {
			log.Printf("Error unmarshaling the message: %v\n", err)
		}

		switch msg.Command {
		case consts.MsgByteCount:
			//TODO: Finish this
			in, out, tin, tout := utils.BytecountToUint(msg.Args["in"], msg.Args["out"],
				msg.Args["tin"], msg.Args["tout"])

			fmt.Println("Got bytecount:", utils.FormatSize(in), utils.FormatSize(out),
				utils.FormatSize(tin), utils.FormatSize(tout))

		case consts.MsgStateChanged:
			state, ok := msg.Args["state"]
			if !ok {
				log.Println("Error: Unknown state message")
				break
			}
			gui.state = state
			//TODO: Update the program status
			fmt.Println("Got state:", state)

		case consts.MsgDisconnected:
			//TODO: Update text
		case consts.MsgError:
			// TODO: Show error
			fmt.Println("Got error")
		}

	}
	select {
	case <-gui.quit:
		log.Println("Closed")

	default:
		if err := scanner.Err(); err != nil && err != io.EOF {
			log.Printf("Read error: %v", err)
		}
	}

	// TODO: Reset everything
	gui.server = nil
	// Wait if the daemon is restarting, then try connecting again
	time.Sleep(500 * time.Millisecond)

	// IMPORTANT: You can't show dialogs in goroutines
	glib.IdleAdd(gui.connectToDaemon)

}

func (gui *mainGUI) connectToDaemon() {
	c, err := net.Dial("unix", consts.UnixSocket)
	if err != nil {

	firstDialog:
		msgDialog := gtk.MessageDialogNew(gui.window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR,
			gtk.BUTTONS_YES_NO, "Vodga daemon is not running, do you want to start it?")
		response := msgDialog.Run()
		msgDialog.Destroy()
		if response == gtk.RESPONSE_YES {
			cmd := exec.Command("systemctl", "start", "vodga.service")
			if err := cmd.Start(); err != nil {
				log.Fatalf("cmd.Run: %v", err)
			}

			if err := cmd.Wait(); err != nil {
				if _, ok := err.(*exec.ExitError); ok {
					msgDialog2 := gtk.MessageDialogNew(gui.window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR,
						gtk.BUTTONS_OK, "Cannot start the daemon")
					msgDialog2.Run()
					msgDialog2.Destroy()

					// Start over
					goto firstDialog

				} else {
					log.Fatalf("cmd.Wait: %v", err)
				}
			}
		} else {
			gui.window.Close()
			return
		}
	}
	gui.server = c
	go gui.listenToDaemon()
}

func (gui *mainGUI) initWidgets() {
	statusIcon, err := gtk_deprecated.StatusIconNewFromIconName("gtk-disconnect")
	if err != nil {
		log.Fatalf("Error: Cannot create tray icon")
	}
	gui.trayIcon = statusIcon

	menu, err := gtk.MenuNew()
	if err != nil {
		log.Fatalf("Error: Cannot create tray menu")
	}
	defer menu.ShowAll()

	gui.trayMenu = menu

	menuItemExit, err := gtk.MenuItemNewWithLabel("Exit")
	if err != nil {
		log.Fatalf("Error: Cannot create tray menu")
	}

	_, _ = menuItemExit.Connect("activate", func() {
		gtk.MainQuit()
	})

	gui.trayMenuItem = menuItemExit
	gui.trayMenu.Append(gui.trayMenuItem)
	_, _ = gui.trayIcon.Connect("activate", func() {
		if !gui.window.IsVisible() {
			gui.window.SetVisible(true)
		} else if !gui.window.IsActive() {
			gui.window.Present()
		} else {
			gui.window.SetVisible(false)
		}
	})

	_, _ = gui.trayIcon.Connect("popup_menu",
		func(icon interface{}, a uint, b uint32) {
			gui.trayIcon.PopupMenu(gui.trayMenu, a, b)
		},
	)
}

func (gui *mainGUI) showMainWindow() {
	if gui.window == nil {
		log.Fatalf("Error: Main window is not initialized")
	}
	gui.window.ShowAll()
}
