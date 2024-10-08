package packet

func ClientClose() []byte {
    return []byte("close")
}

func IsEmpty(data []byte) bool {
    return len(data) == 0
}

func IsClientClosed(data []byte) bool {
    return string(data) == "close"
}

