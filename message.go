package main


func getPingMsg() []byte {
	pingMsg := make([]byte, 5)
	copy(pingMsg[0:4], []byte{0, 0, 0, 1})
	copy(pingMsg[4:5], []byte{0})
	return pingMsg
}

func getPongMsg() []byte {
	pingMsg := make([]byte, 5)
	copy(pingMsg[0:4], []byte{0, 0, 0, 1})
	copy(pingMsg[4:5], []byte{1})
	return pingMsg
}

func getMsgType(msg []byte) string {
	availableMsgTypes := map[byte]string{
		0: "ping",
		1: "pong",
		2: "chat",
		3: "file_info",
		4: "file_accept",
		5: "file_data",
	}
	msgType := availableMsgTypes[msg[0]]
	return msgType
}
