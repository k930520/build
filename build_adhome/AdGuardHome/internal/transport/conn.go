package transport

import (
	"bytes"
	"net"
	"strings"

	utls "github.com/sardanioss/utls"
	"golang.org/x/net/publicsuffix"
)

const (
	recordHeaderLen           = 5
	recordTypeHandshake uint8 = 22
)

var recordHeader = []byte{recordTypeHandshake, 0x03, 0x04, 0x00, 0x00}

type recordHandshakeFragConn struct {
	net.Conn
}

func (c *recordHandshakeFragConn) Write(b []byte) (int, error) {
	if b[0] == recordTypeHandshake && len(b) == recordHeaderLen+(int(b[3])<<8|int(b[4])) {
		if hello := utls.UnmarshalClientHello(b[recordHeaderLen:]); hello != nil && hello.ServerName != "" {
			serverName := hello.ServerName
			serverNameStartIndex := bytes.Index(b, []byte(serverName))
			if serverNameStartIndex >= recordHeaderLen {
				//header := make([]byte, recordHeaderLen)
				//header[0] = recordTypeHandshake
				//vers := tls.VersionTLS13
				//header[1] = byte(vers >> 8)
				//header[2] = byte(vers)
				suffix, _ := publicsuffix.PublicSuffix(serverName)
				i := len(serverName) - len(suffix) - 1
				eTLDStartIndex := i + serverNameStartIndex
				eSLDStartIndex := strings.LastIndex(serverName[:i], ".") + serverNameStartIndex
				helloBytes := make([]byte, 0)
				if serverNameStartIndex < eSLDStartIndex {
					makeRecordAndAppend(recordHeader, b[recordHeaderLen:eSLDStartIndex], &helloBytes)
					makeRecordAndAppend(recordHeader, b[eSLDStartIndex:eTLDStartIndex], &helloBytes)
					makeRecordAndAppend(recordHeader, b[eTLDStartIndex:], &helloBytes)
				} else {
					makeRecordAndAppend(recordHeader, b[recordHeaderLen:eTLDStartIndex], &helloBytes)
					makeRecordAndAppend(recordHeader, b[eTLDStartIndex:], &helloBytes)
				}
				n, err := c.Conn.Write(helloBytes)
				if err != nil {
					return n, err
				}
				return len(b), nil
			}
		}
	}
	return c.Conn.Write(b)
}

func makeRecordAndAppend(header, payload []byte, result *[]byte) {
	header[3] = byte(len(payload) >> 8)
	header[4] = byte(len(payload))
	*result = append(*result, header...)
	*result = append(*result, payload...)
}
