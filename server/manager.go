package server

type ClientManager struct {
	//map[clientID]client
	Client map[string]*ClientCtrl
}

func NewClientManager() (cm *ClientManager) {
	cm = &ClientManager{
		Client: make(map[string]*ClientCtrl),
	}
	return

}

func (cm *ClientManager) Add(clientId string, clientCtrl *ClientCtrl) {
	cm.Client[clientId] = clientCtrl
}

type ProxyManager struct {
}

func NewProxyManager() (pm *ProxyManager) {
	pm = &ProxyManager{}
	return

}
