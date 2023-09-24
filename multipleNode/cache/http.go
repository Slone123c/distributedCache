package cache

import (
	"fmt"
	"geecache/consistentHash"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_cache/"
	defaultReplicas = 50
)

type httpGetter struct {
	baseURL string
}

type HTTPPool struct {
	self        string
	basePath    string
	mu          sync.Mutex
	nodes       *consistentHash.Map
	httpGetters map[string]*httpGetter
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	// 构建完整的 URL，包括 baseURL, group 和 key
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	// 发送 HTTP GET 请求
	res, err := http.Get(u)
	if err != nil {

		return nil, err
	}
	// 确保关闭响应体
	defer res.Body.Close()

	// 检查 HTTP 状态码
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}
	// 读取响应体的全部内容
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}
	// 返回响应体的字节表示
	return bytes, nil

}

// 确保 httpGetter 实现了 NodeGetter 接口
var _ NodeGetter = (*httpGetter)(nil)

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServerHTTP 实现 http.Handler 接口
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.Log("Received request : %s", r.URL.Path)
	//p.Log("%s", r.URL.Path)
	p.Log("p.basePath:%s", p.basePath)
	// 验证请求路径是否符合预期
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	// 打印请求信息
	p.Log("%s %s", r.Method, r.URL.Path)

	// 解析 URL，提取组名和键
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	fmt.Println("url:", r.URL)
	fmt.Println("url.path:", r.URL.Path)
	fmt.Println("r.URL.Path[len(p.basePath):]:", r.URL.Path[len(p.basePath):])
	fmt.Println("parts:", parts)

	groupName := parts[0] // 缓存组名称
	key := parts[1]       // 缓存键

	// 获取缓存组
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 获取键对应的缓存数据
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回缓存数据
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// Set 用于设置存储节点和对应的 httpGetter
// httpGetter 作为 HTTP 客户端用于与远程缓存节点进行通信
func (p *HTTPPool) Set(nodes ...string) {
	// 加锁，保证多线程安全
	p.mu.Lock()
	// 释放锁，使用 defer 确保锁会被释放
	defer p.mu.Unlock()
	// 初始化一致性哈希算法的 Map，用于节点分布
	p.nodes = consistentHash.NewMap(defaultReplicas, nil)
	// 将传入的 nodes（节点）添加到一致性哈希环
	p.nodes.Add(nodes...)
	// 初始化 httpGetters，这个 map 用于存储每个节点对应的 httpGetter 结构体
	p.httpGetters = make(map[string]*httpGetter, len(nodes))
	for _, node := range nodes {
		// httpGetter 结构体用于对该节点进行 HTTP 请求。
		// baseURL 是每个节点 HTTP 服务的基础 URL。
		p.httpGetters[node] = &httpGetter{baseURL: node + p.basePath}
	}
}

// SelectNode 用于在分布式环境中选择一个合适节点
func (p *HTTPPool) SelectNode(key string) (NodeGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 调用 p.nodes.Get(key) 获取应该处理这个键的节点地址
	if node := p.nodes.Get(key); node != "" && node != p.self {
		p.Log("Select node %s", node)
		// 返回该节点的HTTP客户端和true
		return p.httpGetters[node], true
	}
	return nil, false
}
