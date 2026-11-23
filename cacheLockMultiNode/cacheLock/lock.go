package cacheLock

import "sync"

// call 表示一个正在进行或者已完成的调用
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// TaskGroup 用来表示一组调用，用于管理请求
type TaskGroup struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do 用于保证由参数 key 标识的任务，在同一时间只会被执行一次，如果有多个请求尝试执行
// 同样的任务，只有其中一个会执行，其他的则等待完成后获取结果
func (g *TaskGroup) Do(key string, fnc func() (interface{},
	error)) (interface{}, error) {
	// g.mu 是保护 Group 的成员变量 m 不被并发读写而加上的锁。
	// 加锁，确保并发安全
	g.mu.Lock()
	// 如果 m 还没有初始化，就进行初始化
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 检查这个特定的任务（key）是否已经在执行
	if c, ok := g.m[key]; ok {
		// 如果是，则解锁并等待该任务完成
		// 如果在等待之前不进行解锁，其他尝试访问该锁的线程或者goroutine
		// 就会被阻塞，可能导致死锁，所以让出资源
		g.mu.Unlock()
		// 让出资源后，等待该任务完成
		c.wg.Wait()
		return c.val, c.err
	}
	// 如果任务没在执行，则初始化一个新的 call，并加入到 map 中
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c // 添加到 g.m，表明 key 已经有对应的请求在处理这个数据
	/*
		接下来的代码将执行用户定义的函数fnc()，这一步可能会花费不确定的时间。
		如果不解锁，那么其他所有尝试访问该TaskGroup对象的Goroutines都将被阻塞
	*/
	g.mu.Unlock()
	// 执行实际的数据处理任务
	c.val, c.err = fnc()
	// 任务已完成
	c.wg.Done()
	// 再次加锁，并从 map 中删除该任务
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
