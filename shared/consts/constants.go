package consts

const (
	UIFilePath    = "/home/alireza/go/src/github.com/TheWeirdDev/Vodga/ui/vodga.ui"
	GeoIPDataBase = "/home/alireza/Downloads/GeoLite2-City.mmdb"
	UnixSocket    = "/tmp/vodgad.sock"
	MgmtSocket    = "/tmp/vodgad_mgmt.sock"
	UnknownCmd    = "UNKNOWN_COMMAND"
)

const (
	AuthNoAuth   = "NO_AUTH"
	AuthUserPass = "AUTH_USER_PASS"
)

const (
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
	MsgAuthFailed   = "AUTH_FAILED"
)

const (
	StateCONNECTED    = "CONNECTED"
	StateCONNECTING   = "CONNECTING"
	StateWAIT         = "WAIT"
	StateAUTH         = "AUTH"
	StateGET_CONFIG   = "GET_CONFIG"
	StateASSIGN_IP    = "ASSIGN_IP"
	StateADD_ROUTES   = "ADD_ROUTES"
	StateRECONNECTING = "RECONNECTING"
	StateEXITING      = "EXITING"
)
