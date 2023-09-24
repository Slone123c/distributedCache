package main

import (
	"flag"
	"fmt"
	"geecache/multipleNode/cache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

var (
	apiAddr = "http://localhost:9999"
	addrMap = map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}
)

func createGroup() *cache.Group {
	return cache.NewGroup("scores", 2<<10, cache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startCacheServer(addr string, addrs []string, group *cache.Group) {
	// 创建一个新的 HTTPPool 实例。HTTPPool 是自定义的HTTP服务器，负责处理缓存相关的HTTP请求。
	// 对于每一个节点，都加入了 3 个缓存节点，这样他们之间就可以通信
	nodes := cache.NewHTTPPool(addr)
	// 使用 Set 方法设置一组服务器地址，用于处理缓存请求。
	nodes.Set(addrs...)
	// 将新创建的 HTTPPool 实例注册到缓存组（Group）中。
	group.RegisterNodes(nodes)
	log.Println("cache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], nodes))
}

func startAPIServer(apiAddr string, group *cache.Group) {
	// http.Handle 注册一个新的路由处理函数，这里是 "/api"
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// 从请求 URL 中获取 "key" 参数
			key := r.URL.Query().Get("key")
			// 使用 cache.Group 的 Get 方法获取该 key 的缓存值
			view, err := group.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// 设置响应头
			w.Header().Set("Content-Type", "application/octet-stream")
			// 将缓存数据写入到 HTTP 响应体中
			w.Write(view.ByteSlice())
		},
	))
	log.Println("api server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	// 从命令行解析端口，默认为8001
	flag.IntVar(&port, "port", 8001, "cache server port")
	// 从命令行解析是否启动API服务器，默认为false
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse() // 解析命令行参数
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	group := createGroup()
	if api {
		go startAPIServer(apiAddr, group)
	}
	startCacheServer(addrMap[port], []string(addrs), group)
}
