package geecache

import (
	"fmt"
	"log"
	"sync"
)

type IGroup interface {
	Get(key string) (ByteView, error)
	RegisterPeers(peers PeerPicker)
}

// A Group is a cache namespace and associated data loaded spread over
// 一个 Group 可以认为是一个缓存的命名空间
// 每个 Group 拥有一个唯一的名称 name。
// 比如可以创建三个 Group，缓存学生的成绩命名为 scores，缓存学生信息的命名为 info，缓存学生课程的命名为 courses
// 第二个属性是 getter Getter，即缓存未命中时获取源数据的回调(callback)
// 第三个属性是 mainCache cache，即一开始实现的并发缓存
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
}

// 定义接口 Getter
type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义函数类型 GetterFunc，并实现 Getter 接口的 Get 方法
type GetterFunc func(key string) ([]byte, error)

// 回调函数 Get(key string)([]byte, error)，参数是 key，返回值是 []byte
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu sync.RWMutex
	//全局变量 groups
	groups = make(map[string]*Group)
)

// NewGroup 创建新的实例Group
// 构建函数 NewGroup 用来实例化 Group，并且将 group 存储在全局变量 groups 中
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
		//panic("nil Getter")
		//这样写报错 查阅说是idea的自身问题 编译可以通过
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// GetGroup 返回之前用NewGroup创建并存储在groups中的group
// 如果不存在返回nil
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 方法实现了上述所说的流程 ⑴ 和 ⑶
// 流程 ⑴ ：从 mainCache 中查找缓存，如果存在则返回缓存值。
// 流程 ⑶ ：缓存不存在，则调用 load 方法
// load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取）
// getLocally 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	return g.load(key)
}

// RegisterPeers registers a PeerPicker for choosing remote peer
// 新增 RegisterPeers() 方法，
// 将实现了PeerPicker接口的HTTPPool注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取）
// getLocally 调用用户回调函数 g.getter.Get() 获取源数据
// 并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）

// 修改 load 方法，使用 PickPeer() 方法选择节点，若非本机节点，
// 则调用 getFromPeer() 从远程获取。若是本机节点或失败，则回退到 getLocally()
func (g *Group) load(key string) (value ByteView, err error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[GeeCache] Failed to get from peer", err)
		}
	}

	return g.getLocally(key)
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err

	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

// 新增 getFromPeer() 方法，
// 使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer PeerClient, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}
