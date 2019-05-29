package utils

// #cgo pkg-config: gdk-3.0 gio-2.0 glib-2.0 gobject-2.0 gtk+-3.0
// #include <glib.h>
import "C"

func FormatSize(size uint64) string {
	return C.GoString((*C.char)(C.g_format_size_full(C.gulong(size), C.G_FORMAT_SIZE_IEC_UNITS)))
}