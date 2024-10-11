package amproxy

import "net"


type AMTCPConnection struct {
    conn net.Conn

    connStr string
}

func CreateTCPConnectionFrom(connString string) (AMConnection, error) {
    conn, err := net.Dial("tcp", connString)
    if err != nil {
        return nil, err
    }

    return &AMTCPConnection{
        conn: conn,
        connStr: connString,
    }, nil
}

func (a *AMTCPConnection) Read(b []byte) (int, error) {
    return a.conn.Read(b)
}

func (a *AMTCPConnection) Write(b []byte) (int, error) {
    return a.conn.Write(b)
}

func (a *AMTCPConnection) Close() error {
    return a.conn.Close()
}

func (a *AMTCPConnection) String() string {
    return a.connStr
}

func (a *AMTCPConnection) Id() string {
    return "AMTCPConnection DOES NOT HAVE ID YET...."
}

func (a *AMTCPConnection) Addr() string {
    return a.connStr
}

