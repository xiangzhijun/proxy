package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	log "github.com/cihub/seelog"
)

// TLS extension numbers
const (
	extensionServerName          uint16 = 0
	extensionStatusRequest       uint16 = 5
	extensionSupportedCurves     uint16 = 10
	extensionSupportedPoints     uint16 = 11
	extensionSignatureAlgorithms uint16 = 13
	extensionALPN                uint16 = 16
	extensionSCT                 uint16 = 18
	extensionSessionTicket       uint16 = 35
	extensionNextProtoNeg        uint16 = 13172 // not IANA assigned
	extensionRenegotiationInfo   uint16 = 0xff01
)

const (
	typeClientHello uint8 = 1
)

type HttpsReverseProxy struct {
	router *Routers
	l      net.Listener
}

func NewHttpsReverseProxy(l net.Listener) (rp *HttpsReverseProxy) {
	rp = &HttpsReverseProxy{
		router: NewRouters(),
		l:      l,
	}

	return

}

func (hp *HttpsReverseProxy) Run() {
	for {
		conn, err := hp.l.Accept()
		if err != nil {
			log.Warn("https proxy server is closed:", err)
			return
		}

		go hp.handler(conn)
	}
}

func (hp *HttpsReverseProxy) Register(domain, url string, pxy Proxy) error {
	r := hp.router.Find(domain, url)
	if r != nil {
		err := fmt.Errorf("Register error:router is existed")
		return err
	} else {
		hp.router.Add(domain, url, pxy)
		log.Debug("add router  ", domain, ":", url)
		return nil
	}
}
func (hp *HttpsReverseProxy) Remove(domain, url string) {
	hp.router.Del(domain, url)
}

func (hp *HttpsReverseProxy) GetConn(domain, url string) (net.Conn, error) {
	r := hp.router.Get(domain, url)
	if r == nil {
		return nil, fmt.Errorf("conn not found")
	}
	return r.pxy.GetWorkConn()
}

func (hp *HttpsReverseProxy) handler(conn net.Conn) {
	defer conn.Close()
	c, host, err := GetHttpsHostName(conn)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug(hostInfo)
	domain := strings.ToLower(host)
	url := "/"
	client_conn, err2 := hp.GetConn(domain, url)
	if err2 != nil {
		log.Error(err)
		return
	}
	defer client_conn.Close()

	BridgeConn(c, client_conn)
}

type CacheConn struct {
	net.Conn
	sync.Mutex
	buff *bytes.Buffer
}

func (c *CacheConn) Read(p []byte) (n int, err error) {
	c.Lock()
	if c.buff == nil {
		c.Unlock()
		return c.Conn.Read(p)
	}
	c.Unlock()
	n, err = c.buff.Read(p)
	if err == io.EOF {
		c.Lock()
		c.buff = nil
		c.Unlock()
		var n1 int
		n1, err = c.Conn.Read(p[n:])
		n += n1
	}
	return
}

func GetHttpsHostName(conn net.Conn) (net.Conn, string, error) {
	cacheConn := &CacheConn{
		Conn: conn,
		buff: bytes.NewBuffer(make([]byte, 0, 1024)),
	}

	dataReader := io.TeeReader(conn, cacheConn.buff)

	host, err := parseHandshake(dataReader)

	return cacheConn, host, err
}

func parseHandshake(rd io.Reader) (host string, err error) {
	data := make([]byte, 1024)
	//origin := data
	length, err := rd.Read(data)
	if err != nil && err != io.EOF {
		return
	} else {
		if length < 47 {
			err = fmt.Errorf("readHandshake: proto length[%d] is too short", length)
			return
		}
	}
	data = data[:length]
	if uint8(data[5]) != typeClientHello {
		err = fmt.Errorf("readHandshake: type[%d] is not clientHello", uint16(data[5]))
		return
	}

	// session
	sessionIdLen := int(data[43])
	if sessionIdLen > 32 || len(data) < 44+sessionIdLen {
		err = fmt.Errorf("readHandshake: sessionIdLen[%d] is long", sessionIdLen)
		return
	}
	data = data[44+sessionIdLen:]
	if len(data) < 2 {
		err = fmt.Errorf("readHandshake: dataLen[%d] after session is short", len(data))
		return
	}

	// cipher suite numbers
	cipherSuiteLen := int(data[0])<<8 | int(data[1])
	if cipherSuiteLen%2 == 1 || len(data) < 2+cipherSuiteLen {
		err = fmt.Errorf("readHandshake: dataLen[%d] after cipher suite is short", len(data))
		return
	}
	data = data[2+cipherSuiteLen:]
	if len(data) < 1 {
		err = fmt.Errorf("readHandshake: cipherSuiteLen[%d] is long", cipherSuiteLen)
		return
	}

	// compression method
	compressionMethodsLen := int(data[0])
	if len(data) < 1+compressionMethodsLen {
		err = fmt.Errorf("readHandshake: compressionMethodsLen[%d] is long", compressionMethodsLen)
		return
	}

	data = data[1+compressionMethodsLen:]
	if len(data) == 0 {
		// ClientHello is optionally followed by extension data
		err = fmt.Errorf("readHandshake: there is no extension data to get servername")
		return
	}
	if len(data) < 2 {
		err = fmt.Errorf("readHandshake: extension dataLen[%d] is too short")
		return
	}

	extensionsLength := int(data[0])<<8 | int(data[1])
	data = data[2:]
	if extensionsLength != len(data) {
		err = fmt.Errorf("readHandshake: extensionsLen[%d] is not equal to dataLen[%d]", extensionsLength, len(data))
		return
	}
	for len(data) != 0 {
		if len(data) < 4 {
			err = fmt.Errorf("readHandshake: extensionsDataLen[%d] is too short", len(data))
			return
		}
		extension := uint16(data[0])<<8 | uint16(data[1])
		length := int(data[2])<<8 | int(data[3])
		data = data[4:]
		if len(data) < length {
			err = fmt.Errorf("readHandshake: extensionLen[%d] is long", length)
			return
		}

		switch extension {
		case extensionRenegotiationInfo:
			if length != 1 || data[0] != 0 {
				err = fmt.Errorf("readHandshake: extension reNegotiationInfoLen[%d] is short", length)
				return
			}
		case extensionNextProtoNeg:
		case extensionStatusRequest:
		case extensionServerName:
			d := data[:length]
			if len(d) < 2 {
				err = fmt.Errorf("readHandshake: remiaining dataLen[%d] is short", len(d))
				return
			}
			namesLen := int(d[0])<<8 | int(d[1])
			d = d[2:]
			if len(d) != namesLen {
				err = fmt.Errorf("readHandshake: nameListLen[%d] is not equal to dataLen[%d]", namesLen, len(d))
				return
			}
			for len(d) > 0 {
				if len(d) < 3 {
					err = fmt.Errorf("readHandshake: extension serverNameLen[%d] is short", len(d))
					return
				}
				nameType := d[0]
				nameLen := int(d[1])<<8 | int(d[2])
				d = d[3:]
				if len(d) < nameLen {
					err = fmt.Errorf("readHandshake: nameLen[%d] is not equal to dataLen[%d]", nameLen, len(d))
					return
				}
				if nameType == 0 {
					serverName := string(d[:nameLen])
					host = strings.TrimSpace(serverName)
					return host, nil
				}
				d = d[nameLen:]
			}
		}
		data = data[length:]
	}
	err = fmt.Errorf("Unknow error")
	return
}
