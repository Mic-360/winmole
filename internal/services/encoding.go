package services

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func stripUTF8BOM(payload []byte) []byte {
	if len(payload) >= len(utf8BOM) && payload[0] == utf8BOM[0] && payload[1] == utf8BOM[1] && payload[2] == utf8BOM[2] {
		return payload[len(utf8BOM):]
	}
	return payload
}
