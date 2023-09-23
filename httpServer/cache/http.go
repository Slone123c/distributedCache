package cache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const defaultBasePath = "/_cache/"

// HTTPPool 实现了一个 HTTP 服务器，用于处理缓存请求
type HTTPPool struct {
	self     string // 当前服务器地址
	basePath string // API 基础路径
}

// NewHTTPPool 创建一个新的 HTTPPool 实例
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 打印日志信息
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServerHTTP 实现 http.Handler 接口
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
