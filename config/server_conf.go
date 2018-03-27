package config

import (
	"github.com/toml"
	"io/ioutil"
)

type ServerConfig struct {
	ServerIP      string `toml:"server_ip"`
	ServerPort    int    `toml:"server_port"`
	UserTokenFile string `toml:"user_token_file"`

	HttpProxy  *HttpProxyConf  `toml:"http_proxy"`
	HttpsProxy *HttpsProxyConf `toml:"https_proxy"`
}

//服务端proxy的配置
type HttpProxyConf struct {
	VisitIP   string `toml:"visit_ip"`
	VisitPort int    `toml:"visit_port"`
}

type HttpsProxyConf struct {
	VisitIP   string `toml:"visit_ip"`
	VisitPort int    `toml:"visit_port"`
}

func NewServerConfWithFile(file_name string) (server_conf *ServerConfig, err error) {
	data, err := ioutil.ReadFile(file_name)
	if err != nil {
		return nil, err
	}

	server_conf = new(ServerConfig)
	_, err = toml.Decode(string(data), server_conf)
	return
}
