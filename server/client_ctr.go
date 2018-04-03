package server

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	msg "proxy/message"
	"proxy/utils"
)

type ClientCtrl struct {
	svr  *Service
	conn net.Conn

	loginMsg *msg.Login
	clientId string
	token    string
	proxies  map[string]Proxy

	sendCh    chan (msg.Message)
	receiveCh chan (msg.Message)

	connPool chan (net.Conn)

	lastPing time.Time
	mu       sync.RWMutex
}

func NewClientCtrl(svr *Service, loginMsg *msg.Login, conn net.Conn, token string) (client *ClientCtrl) {
	client = &ClientCtrl{
		svr:       svr,
		conn:      conn,
		loginMsg:  loginMsg,
		clientId:  loginMsg.ClientId,
		token:     token,
		proxies:   make(map[string]Proxy),
		sendCh:    make(chan msg.Message, 10),
		receiveCh: make(chan msg.Message, 10),
		connPool:  make(chan net.Conn, loginMsg.ConnPoolCount+10),
		lastPing:  time.Now(),
	}
	return

}

func (c *ClientCtrl) Start() {
	loginResp := msg.LoginResp{
		ClientId: c.clientId,
		Status:   1,
	}

	if err := msg.WriteMsg(msg.TypeLoginResp, loginResp, c.conn); err != nil {
		log.Error(err)
		return
	}

	go c.manager()

	for i := 0; i < c.loginMsg.ConnPoolCount; i++ {
		c.ReqNewWorkConn()
	}
	return
}

func (c *ClientCtrl) manager() {
	go c.readMsg()
	go c.writeMsg()

	pingCheck := time.NewTicker(time.Second)
	defer pingCheck.Stop()

	for {
		select {
		case <-pingCheck.C:
			if time.Since(c.lastPing) > time.Duration(c.svr.conf.PingTimeout)*time.Second {
				log.Error("client ping timeout")
				c.conn.Close()
				return
			}

		case rawMsg, ok := <-c.receiveCh:
			if !ok {
				c.conn.Close()
				return
			}
			msg_type, m, err := msg.UnPack(rawMsg)
			if err != nil {
				log.Error(err)
				c.conn.Close()
				return
			}
			switch msg_type {
			case msg.TypeNewProxy:
				newProxy := m.(msg.NewProxy)
				c.RegisterProxy(newProxy)
			case msg.TypePing:
				c.lastPing = time.Now()
				log.Debug("receive ping msg from client:", c.clientId)
				pong := msg.Pong{}
				m, err := msg.Pack(msg.TypePong, pong)
				if err != nil {
					log.Error(err)
					continue
				}
				c.sendCh <- m
			}
		}

	}
}

func (c *ClientCtrl) readMsg() {
	conn := utils.NewReader(c.conn, []byte(c.token))

	for {
		if m, err := msg.ReadRawMsg(conn); err != nil {
			if err == io.EOF {
				log.Debug("read message from server EOF")
				return

			} else {
				log.Error(err)
				return
			}
		} else {
			c.receiveCh <- m
		}

	}

}

func (c *ClientCtrl) writeMsg() {
	conn, err := utils.NewWriter(c.conn, []byte(c.token))
	if err != nil {
		log.Error(err)
		c.conn.Close()
		return
	}

	for {
		if m, ok := <-c.sendCh; !ok {
			log.Error("send message chan closed")
			return
		} else {
			if err := msg.WriteRawMsg(m, conn); err != nil {
				log.Error(err)
				return
			}

		}

	}

}

func (c *ClientCtrl) NewWorkConn(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Error("register work connection error:", err)
		}
	}()

	select {
	case c.connPool <- conn:
		log.Info("Add new work connection to ConnPool.[ClientId]:", c.clientId)
	default:
		log.Info("Add new wockConn failed,ConnPool is full.[ClientId]:", c.clientId)
		c.Close()
	}

}

func (c *ClientCtrl) GetWorkConn() (conn net.Conn, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(r)
		}
	}()

	var ok bool
	select {
	case conn, ok = <-c.connPool:
		if !ok {
			err = fmt.Errorf("connPool is closed")
			return
		}
		log.Debug("get conn from pool")
	default:
		c.ReqNewWorkConn()
		select {
		case conn, ok = <-c.connPool:
			if !ok {
				err = fmt.Errorf("connPool is closed")
				return
			}
			log.Debug("get work connection from pool")

		case <-time.After(time.Duration(10) * time.Second):
			err = fmt.Errorf("get new work connection timeout")
			log.Warn(err)
			return
		}
	}

	c.ReqNewWorkConn()
	return

}

func (c *ClientCtrl) ReqNewWorkConn() {
	defer func() {
		if r := recover(); r != nil {
			log.Error(r)
		}
	}()
	m := msg.ReqWorkConn{}
	M, err := msg.Pack(msg.TypeReqWorkConn, m)
	if err != nil {
		log.Error(err)
		return
	}

	c.sendCh <- M

	return

}

func (c *ClientCtrl) RegisterProxy(m msg.NewProxy) {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	pxy := NewProxy(c, m)

	pxy.Run()

	c.mu.Lock()
	c.proxies[pxy.GetName()] = pxy
	c.mu.Unlock()

	resp := msg.NewProxyResp{
		ProxyName: pxy.GetName(),
	}

	M, err := msg.Pack(msg.TypeNewProxyResp, resp)
	if err != nil {
		pxy.Close()
		log.Error(err)
		return
	}
	c.sendCh <- M
}

func (c *ClientCtrl) Close() {
	c.conn.Close()
}
