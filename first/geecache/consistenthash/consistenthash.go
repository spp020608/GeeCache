package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map constains all hashed keys
type Map struct {
	hash     Hash           //默认为 crc32.ChecksumIEEE 算法
	replicas int            //虚拟节点倍数
	keys     []int          // 哈希环
	hashMap  map[int]string //虚拟节点与真实节点的映射表 hashMap，键是虚拟节点的哈希值，值是真实节点的名称。
}

// New 创建Map实例
func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add adds some keys to the hash.
// Add 函数允许传入 0 或 多个真实节点的名称。
// 对每一个真实节点 key，对应创建 m.replicas 个虚拟节点
// 虚拟节点的名称是：strconv.Itoa(i) + key，即通过添加编号的方式区分不同虚拟节点。
// 使用 m.hash() 计算虚拟节点的哈希值，使用 append(m.keys, hash) 添加到环上。
// 在 hashMap 中增加虚拟节点和真实节点的映射关系。
// 最后一步，环上的哈希值排序
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Remove use to remove a key and its virtual keys on the ring and map
func (m *Map) Remove(key string) {
	for i := 0; i < m.replicas; i++ {
		hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
		idx := sort.SearchInts(m.keys, hash)
		m.keys = append(m.keys[:idx], m.keys[idx+1:]...)
		delete(m.hashMap, hash)
	}
}

// Get gets the closest item in the hash to the provided key.
// 选择节点就非常简单了，第一步，计算 key 的哈希值。
// 第二步，顺时针找到第一个匹配的虚拟节点的下标 idx，
// 从 m.keys 中获取到对应的哈希值。如果 idx == len(m.keys)，
// 说明应选择 m.keys[0]，因为 m.keys 是一个环状结构，所以用取余数的方式来处理这种情况。
// 第三步，通过 hashMap 映射得到真实的节点。
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// Binary search for appropriate replica.
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	//因此，为了确保程序的正确性，我们需要使用取余操作将 idx 映射到 m.keys 的有效索引范围内。
	//具体来说，我们可以使用 idx % len(m.keys) 来得到在循环使用 m.keys 的情况下与 idx 相对应的索引，
	//从而正确地获取对应键的实际值。
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
