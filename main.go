package main

import (
	"Vodga/ui"
	"github.com/gotk3/gotk3/gtk"
)


func main() {
	gtk.Init(nil)

	ui.StartGui()

	gtk.Main()
}

