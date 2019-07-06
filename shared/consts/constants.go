package consts

const (
	//TODO: Change These!
	UIFilePath    = "/home/alireza/go/src/github.com/TheWeirdDev/Vodga/ui/data/vodga.ui"
	AddSingelUI   = "/home/alireza/go/src/github.com/TheWeirdDev/Vodga/ui/data/add_single.ui"
	GeoIPDataBase = "/home/alireza/Downloads/GeoLite2-Country.mmdb"
	UnixSocket    = "/tmp/vodgad.sock"
	MgmtSocket    = "/tmp/vodgad_mgmt.sock"
	UnknownCmd    = "UNKNOWN_COMMAND"
	IPRegex       = "^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
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
