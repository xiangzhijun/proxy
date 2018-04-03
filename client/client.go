package client

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"proxy/config"
	msg "proxy/message"
	"proxy/utils"
)

const (
	ReadTimeout time.Duration = 10 * time.Second
)

type Client struct {
	conn    net.Conn
	config  *config.ClientConfig
	manager *Manager

	clientId string
	Token    string

	sendCh    chan (msg.Message)
	receiveCh chan (msg.Message)
	blocked   chan int
	exit      bool
	lastPong  time.Time

	mu sync.RWMutex
}

func NewClient(conf *config.ClientConfig) (client *Client) {
	client = &Client{
		config:    conf,
		sendCh:    make(chan msg.Message, 10),
		receiveCh: make(chan msg.Message, 10),
		blocked:   make(chan int),
		Token:     conf.Token,
		exit:      false,
	}

	client.manager = NewManager(client, conf.AllProxy, client.sendCh)

	return
}

func (c *Client) Run() {
	for {
		err := c.login()
		if err != nil {
			log.Error(err)
			time.Sleep(10 * time.Second)
		} else {
			break
		}

	}
	log.Info("client closed")
	go c.worker()
}

func (c *Client) login() error {
	if c.conn != nil {
		c.conn.Close()
	}

	now := time.Now().Unix()
	_, sign := utils.GetMD5([]byte(fmt.Sprintf("%s%d", c.config.Token, now)))
	log.Debug(sign)
	loginMsg := msg.Login{
		User:      c.config.User,
		Sign:      sign,
		Timestamp: now,
		ClientId:  c.clientId,
	}

	log.Debug(loginMsg)
	conn, err := c.ConnectToServer()
	if err != nil {
		return err
	}

	log.Debug("connect server success")

	conn.SetDeadline(time.Now().Add(ReadTimeout))
	err = msg.WriteMsg(msg.TypeLogin, loginMsg, conn)
	if err != nil {
		return err
	}

	msg_type, m, err := msg.ReadMsg(conn)
	if err != nil {
		return err
	}
	if msg_type != msg.TypeLoginResp {
		return fmt.Errorf("The response message is not LoginResp")
	}

	loginResp := m.(*msg.LoginResp)
	if loginResp.Error != "" {
		return fmt.Errorf("%s", loginResp.Error)
	}
	conn.SetReadDeadline(time.Time{})

	log.Debug(loginResp)
	c.clientId = loginResp.ClientId
	return nil
}

func (c *Client) worker() {
	go c.readMsg()
	go c.writeMsg()
	go c.msgHandler()

	for {
		select {

		case _, ok := <-c.blocked:
			if !ok {
				close(c.sendCh)
				close(c.receiveCh)

				if c.exit {
					return
				}

				for {
					log.Info("reconnect to server")
					err := c.login()
					if err != nil {
						log.Error("reconnect to server failed:", err)
						time.Sleep(10)
						continue
					}
					break
				}

				c.receiveCh = make(chan msg.Message, 10)
				c.sendCh = make(chan msg.Message, 10)
				c.blocked = make(chan int)

				c.lastPong = time.Now()
			}

		}

	}

}

func (c *Client) msgHandler() {

	c.lastPong = time.Now()
	PingSend := time.NewTicker(time.Duration(c.config.PingInterval) * time.Second)
	defer PingSend.Stop()

	PongCheck := time.NewTicker(time.Second)
	defer PongCheck.Stop()

	for {
		select {
		case <-PingSend.C:
			p := msg.Ping{}
			m, err := msg.Pack(msg.TypePing, p)
			if err != nil {
				log.Error(err)
				continue
			}

			c.sendCh <- m
			log.Debug("send heartbeat to server")

		case <-PongCheck.C:
			if time.Since(c.lastPong) > time.Duration(c.config.PongTimeout)*time.Second {
				log.Error("heartbeat timeout")
				c.conn.Close()
				return
			}
		case M, ok := <-c.receiveCh:
			if !ok {
				return
			}
			msg_type, m, err := msg.UnPack(M)
			if err != nil {
				log.Error(err)
				continue
			}
			switch msg_type {
			case msg.TypeNewProxyResp:
				newProxyResp := m.(msg.NewProxyResp)
				if newProxyResp.Error != "" {
					log.Error("Regsiter new proxy error:", newProxyResp.Error)
					continue
				}

				c.manager.StartProxy(newProxyResp.ProxyName, newProxyResp.RemotePort)
			case msg.TypeReqWorkConn:
				reqWorkConn := m.(msg.ReqWorkConn)
				go c.NewWorkConn(reqWorkConn)
			case msg.TypePong:

				c.lastPong = time.Now()

			}

		}

	}

}

func (c *Client) readMsg() {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	defer close(c.blocked)

	conn := utils.NewReader(c.conn, []byte(c.config.Token))

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

func (c *Client) writeMsg() {
	conn, err := utils.NewWriter(c.conn, []byte(c.config.Token))
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

func (c *Client) NewWorkConn(rm msg.ReqWorkConn) {
	workConn, err := c.ConnectToServer()
	if err != nil {
		log.Error("connect to server error:", err)
		return
	}

	m := msg.NewWorkConn{
		ClientId: c.clientId,
	}

	err = msg.WriteMsg(msg.TypeNewWorkConn, m, workConn)
	if err != nil {
		log.Error("send NewWorkCOnn msg to server error:", err)
		workConn.Close()
		return
	}

	msg_type, sm, err2 := msg.ReadMsg(workConn)
	if err2 != nil {
		log.Error("read StartWork msg error:", err2)
		return
	}
	if msg_type != msg.TypeStartWork {
		log.Error("msg type is not StartWork")
		return
	}

	c.manager.ProxyWork(sm.(msg.StartWork).ProxyName, workConn)
}

func (c *Client) ConnectToServer() (net.Conn, error) {
	server_addr := fmt.Sprintf("%s:%d", c.config.ServerIP, c.config.ServerPort)
	tcp_addr, err := net.ResolveTCPAddr("tcp", server_addr)
	if err != nil {
		return nil, err
	}
	return net.DialTCP("tcp", nil, tcp_addr)
}
