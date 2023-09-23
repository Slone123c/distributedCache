package main

import (
	"fmt"
	"geecache/httpServer/cache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	// 创建一个新组，用于缓存查询结果。这里的 cacheBytes 是 2<<10（也就是 2048 字节，或 2 KB）
	cache.NewGroup("scores", 2<<10, cache.GetterFunc(
		func(key string) ([]byte, error) {
			// 打印日志，指示查询数据库
			log.Println("[SlowDB] search key", key)
			// 从数据库（这里是模拟的 db）中查询
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			// 如果查询不到，返回错误
			return nil, fmt.Errorf("%s not exist", key)
		}))

	// HTTP 服务器的地址
	addr := "localhost:9999"

	// 创建一个 HTTP 池，并绑定服务器地址
	peers := cache.NewHTTPPool(addr)

	// 打印日志，指示服务器运行地址
	log.Println("geecache is running at", addr)

	// 启动 HTTP 服务器
	log.Fatal(http.ListenAndServe(addr, peers))
}
