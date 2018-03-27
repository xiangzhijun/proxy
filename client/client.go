package client

import (
	"fmt"
	"net"
	"time"

	"proxy/config"
	msg "proxy/message"
	"proxy/utils"
)

const (
	ReadTimeout time.Duration = 10 * time.Second
)

type Client struct {
	conn    *net.Conn
	config  *config.ClientConfig
	manager *Manager

	clientId string

	sendCh    chan (msg.Message)
	receiveCh chan (msg.Message)
	lastPong  time.Time
}

func NewClient(conf *config.ClientConfig) (client *Client) {
	client = &Client{
		config:    conf,
		sendCh:    make(chan msg.Message, 10),
		receiveCh: make(chan msg.Message, 10),
	}

	client.manager = NewManager(client, conf.AllProxy, client.sendCh)

	return
}

func (c *Client) login() error {
	if c.conn != nil {
		c.conn.Close()
	}

	now := time.Now().Unix()
	sign := utile.GetMD5([]byte(c.config.Token + now))

	loginMsg := msg.Login{
		User:      c.config.User,
		Sign:      string(sign),
		Timestamp: now,
		CliendId:  c.clientId,
	}

	conn, err := c.ConnectToServer()
	if err != nil {
		return err
	}

	conn.SetDeadline(time.Now().Add(ReadTimeout))
	err = msg.WriterMsg(msg.TypeLogin, loginMsg, conn)
	if err != nil {
		return err
	}

	msg_type, loginResp, err := msg.ReadMsg(conn)
	if err != nil {
		return err
	}
	if msg_type != msg.TypeLoginResp {
		return fmt.Errorf("The response message is not LoginResp")
	}
	if loginResp.Error != nil {
		return fmt.Errorf("%s", loginResp.Error)
	}
	conn.SetReadDeadline(time.Time{})

	c.clientId = loginResp.ClientID
	return nil
}

func (c *Client) ConnectToServer() (net.Conn, error) {
	server_addr := c.config.ServerIP + ":" + c.config.ServerPort
	tcp_addr, err := net.ResolveTCPAddr("tcp", server_addr)
	if err != nil {
		return nil, err
	}

	return net.DialTCP("tcp", nil, tcp_addr)
}
