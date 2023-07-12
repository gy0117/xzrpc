package registry

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultPath    = "/_xzrpc_/registry"
	defaultTimeout = time.Minute * 5
)

// Registry 注册中心
type Registry struct {
	//servers map[string]*ServerItem
	apps    map[string]*App // <appid-env, App>
	lock    sync.RWMutex
	timeout time.Duration
}

func NewRegistry(timeout time.Duration) *Registry {
	return &Registry{
		//servers: make(map[string]*ServerItem),
		apps:    make(map[string]*App),
		timeout: timeout,
	}
}

// Register 注册服务
func (r *Registry) Register(item *ServerItem) (*App, error) {
	key := getKey(item.AppId, item.Env)
	r.lock.RLock()
	app, ok := r.apps[key]
	r.lock.RUnlock()
	if !ok {
		app = NewApp(item.AppId)
	}
	app.addServer(item)

	r.lock.Lock()
	r.apps[key] = app
	r.lock.Unlock()

	log.Printf("registry | Register success, and app: %+v\n", app)

	return app, nil
}

// GetServer 获取服务
func (r *Registry) GetServer(appid, env string) ([]*ServerItem, error) {
	app, ok := r.getApp(appid, env)
	if !ok {
		return nil, ErrNotFoundServerItem
	}
	return app.getServers()
}

// 添加服务实例
func (r *Registry) putServer(addr string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	//item, ok := r.servers[addr]
	//if !ok {
	//	item = NewServerItem()
	//	//r.servers[addr] = &ServerItem{
	//	//	Addr:  addr,
	//	//	start: time.Now(),
	//	//}
	//	r.servers[addr] = item
	//} else {
	//	// if exists, update start time to keep alive
	//	//server.start = time.Now()
	//}
	//item.regTimestamp = time.Now().Unix()

	log.Println("xzrpc registry | register Server success, and addr: ", addr)

}

// 返回可用的服务列表
func (r *Registry) getAliveServers() []string {
	r.lock.Lock()
	defer r.lock.Unlock()

	aliveServers := make([]string, 0)

	//for addr, server := range r.servers {
	//	// 没有超时概念 || 还未超时
	//	if r.timeout == 0 || time.Duration(time.Now().Unix()-server.regTimestamp) <= r.timeout {
	//		aliveServers = append(aliveServers, addr)
	//	} else {
	//		delete(r.servers, addr)
	//	}
	//
	//	//
	//	//if r.timeout == 0 || server.start.Add(r.timeout).After(time.Now()) {
	//	//	aliveServers = append(aliveServers, addr)
	//	//} else {
	//	//	delete(r.servers, addr)
	//	//}
	//}
	return aliveServers
}

// 简单实现
// GET：返回可用的服务列表
// POST：添加服务实例或者发送心跳
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":

		// TODO 后续抽出来
		servers, err := r.GetServer(TEST_APPID, TEST_ENV)
		if err != nil {
			log.Println("registry | ServeHTTP | failed to GetServer: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		aliveServers := make([]string, 0)

		for _, server := range servers {
			if r.timeout == 0 || time.Duration(time.Now().Unix()-server.regTimestamp) <= r.timeout {
				aliveServers = append(aliveServers, server.Address)
			} else {
				// TODO 这里不要删除，感觉不应该在这里处理
				// 1. 删除app中的items
				// 2. 如果app中的items为空，删除registry中的apps
				//delete(r.servers, addr)
			}
		}

		//w.Header().Set("X-Xzrpc-Servers", strings.Join(r.getAliveServers(), ","))
		w.Header().Set("X-Xzrpc-Servers", strings.Join(aliveServers, ","))

	case "POST":
		addr := req.Header.Get("X-Xzrpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// TODO 这个注册正在优化
		//r.putServer(addr)

		item := &ServerItem{
			Address:      addr,
			Env:          TEST_ENV,
			AppId:        TEST_APPID,
			Hostname:     TEST_HOST_NAME,
			regTimestamp: time.Now().Unix(),
		}
		_, err := r.Register(item)
		if err != nil {
			log.Println("registry | ServeHTTP | failed to Register: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *Registry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Printf("xzrpc registry | registry path: %s\n", registryPath)
}

func (r *Registry) getApp(appid, env string) (*App, bool) {
	key := getKey(appid, env)
	r.lock.RLock()
	defer r.lock.RUnlock()
	app, ok := r.apps[key]
	return app, ok
}

func getKey(appId, env string) string {
	return fmt.Sprintf("%s-%s", appId, env)
}
