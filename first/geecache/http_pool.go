package geecache

import (
	"fmt"
	"geecache/geecache/consistenthash"
	"log"
	"net/http"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	//用来记录自己的地址，包括主机名/IP 和端口
	self string
	//basePath，作为节点间通讯地址的前缀，默认是 /_geecache/，那么
	basePath string
	mu       sync.Mutex // guards peers and httpGetters
	//新增成员变量 peers，类型是一致性哈希算法的 Map，用来根据具体的 key 选择节点
	peers *consistenthash.Map
	//新增成员变量 httpGetters，映射远程节点与对应的 httpGetter。
	//每一个远程节点对应一个 httpGetter，因为 httpGetter 与远程节点的地址 baseURL 有关
	httpClient map[string]PeerClient // keyed by e.g. "http://10.0.0.2:8008"
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//判断访问路径的前缀是否是 basePath，不是返回错误。
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	//<groupname>/<key> required 将这部分一分为二
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

type httpGetter struct {
	baseURL string
}

// Set updates the pool's list of peers.
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpClient = make(map[string]PeerClient, len(peers))
	for _, peer := range peers {
		//并为每一个节点创建了一个 HTTP 客户端 httpGetter
		p.httpClient[peer] = NewHttpClient(peer + p.basePath)
	}
}

// PickPeer picks a peer according to key
// PickerPeer() 包装了一致性哈希算法的 Get() 方法，
// 根据具体的 key，选择节点，返回节点对应的 HTTP 客户端
// PickPeer 知道缓存的key,找到主机节点
func (p *HTTPPool) PickPeer(key string) (peer PeerClient, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if hostName := p.peers.Get(key); hostName != "" && hostName != p.self {
		p.Log("Pick PeerHostName: %s", hostName)
		return p.httpClient[hostName], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)
