package consts

const (
	UIFilePath      = "/home/alireza/go/src/github.com/TheWeirdDev/Vodga/ui/vodga.ui"
	UnixSocket      = "/tmp/vodgad.sock"
	MgmtSocket      = "/tmp/vodgad_mgmt.sock"
	UnknownCmd      = "UNKNOWN_COMMAND"
	MsgKilled       = "KILLED"
	MsgStop         = "STOP_SERVER"
	MsgConnect      = "CONNECT"
	MsgError        = "ERROR"
	MsgLog          = "LOG"
	MsgDisconnect   = "DISCONNECT"
	MsgDisconnected = "DISCONNECTED"
	MsgKillOpenvpn  = "KILL_OPENVPN"
	MsgStateChanged = "STATE_CHANGED"
	MsgGetBytecount = "GET_BYTECOUNT"
	MsgByteCount    = "BYTECOUNT"
	AuthNoAuth      = "NO_AUTH"
	AuthUserPass    = "AUTH_USER_PASS"
)
