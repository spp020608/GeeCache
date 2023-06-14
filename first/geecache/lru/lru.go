package lru

import "container/list"

type Cache struct {
	maxBytes  int64      //允许使用的最大内存
	nowBytes  int64      //当前已使用的内存
	ll        *list.List //双线链表
	cache     map[string]*list.Element
	OnEvicted func(key string, value Value) //某条记录被移除时的回调函数，可以为 nil
}

// 双向链表节点的数据类型
type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get look ups a key's value
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)    //移动到双向链条最前面
		kv := ele.Value.(*entry) //获取值
		return kv.value, true
	}
	return
}

// RemoveOldest 删除最旧未用的链条
// c.ll.Back() 取到队首节点，从链表中删除。
// delete(c.cache, kv.key)，从字典中 c.cache 删除该节点的映射关系。
// 更新当前所用的内存 c.nowBytes。
// 如果回调函数 OnEvicted 不为 nil，则调用回调函数。
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() //查询到最后一个元素
	if ele != nil {
		c.ll.Remove(ele) //删除最后一个元素
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nowBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add adds a value to the cache.
// 如果键存在，则更新对应节点的值，并将该节点移到队尾。
// 不存在则是新增场景，首先队尾添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。
// 更新 c.nowBytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点。
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nowBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value}) //添加最新
		c.cache[key] = ele
		c.nowBytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nowBytes {
		c.RemoveOldest()
	}
}

// Len 用来获取添加了多少条数据。
func (c *Cache) Len() int {
	return c.ll.Len()
}
