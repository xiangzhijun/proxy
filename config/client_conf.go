package config

import (
	"github.com/toml"
	"io/ioutil"
)

type ClientConfig struct {
	ServerIP      string       `toml:"server_ip"`
	ServerPort    int          `toml:"server_port"`
	User          string       `toml:"user"`
	Token         string       `toml:"token"`
	PingInterval  int          `toml:"ping_interval"`
	PongTimeout   int          `toml:"pong_timeout"`
	ConnPoolCount int          `toml:"conn_pool_count"`
	AllProxy      []*ProxyConf `toml:"proxy"`
}

//所以客户端proxy的配置
type ProxyConf struct {
	Name       string `toml:"name"`
	Type       string `toml:"type"`
	Encryption bool   `toml:"encryption"`

	LocalIP    string `toml:"local_ip"`
	LocalPort  int    `toml:"local_port"`
	RemotePort int    `toml:"remote_port"`

	Domain string `toml:"domain"`
	Url    string `toml:"url"`
}

func NewClientConfWithFile(file_name string) (client_conf *ClientConfig, err error) {
	data, err := ioutil.ReadFile(file_name)
	if err != nil {
		return nil, err
	}

	client_conf = new(ClientConfig)
	_, err = toml.Decode(string(data), client_conf)
	return
}
