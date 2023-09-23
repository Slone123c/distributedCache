package cache

import (
	"fmt"
	"log"
	"sync"
)

type Group struct {
	name      string
	getter    Getter // 获取数据接口
	mainCache cache
}

// Getter Getter接口定义了一个Get方法，
// 任何实现了Get(key string) ([]byte, error)这个方法的结构体都会自动满足这个接口
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 一个适配器，允许普通函数实现Getter接口
// GetterFunc 是一个函数类型，它接受一个string类型的key并返回一个[]byte和一个error
type GetterFunc func(key string) ([]byte, error)

// Get 为这个函数类型添加了一个方法Get
func (f GetterFunc) Get(key string) ([]byte, error) {
	/*
		注意，这里的Get方法就是调用了f函数本身。
		这样做的结果是，任何符合GetterFunc类型（也就是任何相同签名的函数）
		都隐式地实现了Getter接口
	*/
	return f(key)
}

// 使用全局变量mu和groups来保存所有创建的缓存组
var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup 用于创建一个新的缓存组
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil getter")
	}
	// 使用互斥锁可以确保在同时尝试创建同名的缓存组时，该函数是线程安全的
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:   name,   // 缓存组名
		getter: getter, // 缓存数据的获取逻辑
		mainCache: cache{ // 主缓存
			cacheBytes: cacheBytes, // 缓存的最大字节数
		},
	}
	// 将新创建的Group实例添加到全局的groups映射中
	groups[name] = g
	// 返回新创建的Group实例
	return g
}

// GetGroup 用于通过名称获取一个缓存组。
func GetGroup(name string) *Group {
	/*
		这里使用的是读锁（RLock），因为这个操作是只读的，
		不会改变groups的状态。多个goroutine可以同时持有读锁，提高并发性能
	*/
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	// 尝试从mainCache中获取key的值
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[Cache] hit")
		return v, nil
	}
	// 如果缓存未命中，则调用load方法来加载数据
	return g.load(key)
}

/*
load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取），
getLocally 调用用户回调函数 g.getter.Get() 获取源数据，
并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
*/
func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
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

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
