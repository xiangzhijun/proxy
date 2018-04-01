package server

import (
	"net"
	"sync"

	log "github.com/cihub/seelog"
	msg "proxy/message"
)

type Proxy interface {
	Run()
	Close()

	GetWorkConn() (conn net.Conn, err error)
}

type BaseProxy struct {
	Name string
	Type string

	clientCtrl *ClientCtrl
	mu         sync.RWMutex
}

func (pxy *BaseProxy) GetWorkConn() (conn net.Conn, err error) {
	c := pxy.clientCtrl

	for i := 0; i < c.loginMsg.ConnPoolCount+1; i++ {
		if conn, err = c.GetWorkConn(); err != nil {
			log.Error("get workConn from pool error:", err)
			return
		}

		m := msg.StartProxy{
			ProxyName: pxy.Name,
		}
		err = msg.WriteMsg(msg.TypeStartProxy, m, conn)

		if err != nil {
			log.Error("workConn send startProxy msg error:", err)
			conn.Close()
		} else {
			break
		}
	}
	return
}
