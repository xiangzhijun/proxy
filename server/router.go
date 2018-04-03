package server

import (
	"sort"
	"strings"
	"sync"
)

type Routers struct {
	RouterMap map[string][]*router

	mu sync.RWMutex
}

type router struct {
	domain string
	url    string

	pxy Proxy
}

func NewRouters() *Routers {
	return &Routers{
		RouterMap: make(map[string][]*router),
	}
}

func (Rs *Routers) Add(domain, url string, pxy Proxy) {
	Rs.mu.Lock()
	defer Rs.mu.Unlock()

	rs, ok := Rs.RouterMap[domain]

	if !ok {
		rs = make([]*router, 0, 1)
	}

	r := &router{
		domain: domain,
		url:    url,
		pxy:    pxy,
	}

	rs = append(rs, r)
	sort.Sort(sort.Reverse(ByUrl(rs)))
	Rs.RouterMap[domain] = rs
}

func (Rs *Routers) Del(domain, url string) {
	Rs.mu.Lock()
	defer Rs.mu.Unlock()

	rs, ok := Rs.RouterMap[domain]
	if !ok {
		return
	}

	for i, r := range rs {
		if r.url == url {
			if len(rs) > i+1 {
				Rs.RouterMap[domain] = append(rs[:i], rs[i+1:]...)
			} else {
				Rs.RouterMap[domain] = rs[:i]
			}
			return
		}
	}
}

func (Rs *Routers) Get(domain, url string) *router {
	Rs.mu.RLock()
	defer Rs.mu.RUnlock()

	rs, ok := Rs.RouterMap[domain]
	if !ok {
		return nil
	}

	for _, r := range rs {
		if strings.HasPrefix(url, r.url) {
			return r
		}
	}
	return nil
}

func (Rs *Routers) Find(domain, url string) *router {
	Rs.mu.RLock()
	defer Rs.mu.RUnlock()

	rs, ok := Rs.RouterMap[domain]
	if !ok {
		return nil
	}

	for _, r := range rs {
		if url == r.url {
			return r
		}
	}
	return nil

}

type ByUrl []*router

func (r ByUrl) Len() int {
	return len(r)
}

func (r ByUrl) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r ByUrl) Less(i, j int) bool {
	return strings.Compare(r[i].url, r[j].url) < 0
}
