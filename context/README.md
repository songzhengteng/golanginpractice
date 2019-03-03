# golang context

## context概念
context 可以翻译为上下文、语境、背景
在编程语言中，context的一个程序运行过程中的相关信息的总称，是一个比较泛化的概率。当程序中的一段代码被计算机加载到内存中并开始执行时，这时我
们可以认为对应时间点计算机的所有状态构成了程序运行的上下文信息，比如执行这段代码的进程环境变量、线程局部变量，当前系统的时间，可用的资源。

知乎上有个关于context的提问：https://www.zhihu.com/question/26387327<br>

在编程语言中的context往往有种更加确切的含义， 比如Spring框架中context就是与应用相关的信息集合，比如配置数据、运行的IoC容器数据等

在golang中， 官方的context实现含义更加狭窄，它代表了为了处理一个任务，比如网络请求，所携带的相关数据。这关些数据以context的概念进行封装，
在goroutine间传递请求相关的元数据；并实现了超时功能，通过context的超时功能，goroutine就能感知到请求请求/超时， 然后主动释放资源。

比如，在 Go http 包的 Server 中，每一个请求在都有一个对应的goroutine去处理。请求处理函数通常会启动额外的goroutine用来访问后端服务，比如
数据库和 RPC 服务。用来处理一个请求的goroutine通常需要访问一些与请求特定的数据，比如终端用户的身份认证信息、验证相关的 token、请求的截止
时间。当一个请求被取消或超时时，所有用来处理该请求的goroutine都应该迅速退出，然后系统才能释放这些goroutine占用的资源。

## golang中的context使用
1. 创建context
context的创建使用context包已经定义的生成函数获取，context不支持代码里新建。可以铜鼓如下两个生成函数生成context：
    - context.Background()
    - context.TODO()

context.Background()产生一个永不超时、永不过期、不含值的空context对象， 它通常使用在程序初始化、main函数、测试或处理入站请求的顶层代码
中

context.TODO()产生一个永不超时、永不过期、不含值的空context对象， 它通常使用在需要使用context，但还没确切context的场景，当不知道当前需
要使用什么context，那就选择TODO context

函数调用链上context需要保持传播性，通过WithCancel、WithDeadline、WithTimeout、WithValue在父context的技术上创建新的子context。从而
形成一棵context树。当父context取消是，以改context为root节点的context子树会全部取消。

1. 使用context
context不应该作为一个结构体变量，如果需要context，则应该作为函数的第一个入参传入，为了便于go工具链分析， context的参数名一般为ctx

在调用一个接受context的函数时，及时运行context实参为nil，也**永远不要**传入nil参数作为context的实参。 如果不确定就传入TODO context。

context指传递与请求相关的元数据，不要传入其他可选的参数。
为了避免context中的key冲突导致请求元数据被覆盖，不用直接向context中存入任何key类型为golang内建类型的key/value， 可以使用类型别名或者自
定义类型作为key。可以通过包内封装的方式写入和读出context

context可以在多个goroutine间传递，context是并发访问安全的。

1. 回收context
context使用中需要回收context， 即调用衍生函数时会返回一个CancelFuc，在程序需要调用该cancelFuc， cancelFunc会解出context树中的绑定关
系，并释放相关资源。

## context代码走读
### context定义
在golang中，context被定义为如下接口类型：
```go
type Context interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
}
```
Deadline(): 返回当前context的超时时间戳，当时间到达时context会由于超时导致取消；第二个布尔变量表明context是否设置了超时，当返回
ok==false时，说明context没有设置超时。

Done()： 返回一个信号通道，当信号通道没有关闭时，表明context未被取消（主动取消或者超时取消）； 在已经取消掉的context上调用Done()会直接返
回一个close的通道；在永不会超时的context上调用Done()会返回nil。

Err()： 解释context被取消的原因。当Done被close掉时返回原因；当Done未被close掉时返回nil。

Value(key interface{}): 返回当前context或父context中绑定的键为key的值，如果不存在则返回nil。

### 两个基础的context
context对象的获取不需要在代码中显示的生成， context包封装了两个获取context的函数：
```go
func Background() Context 
func TODO() Context
```
Background() : 返回一个永不超时，永不过期，没有值的context对象，它一般用在程序的main函数中，作为整个程序其它组件的参数。

TODO()： 返回一个永不超时，永不过去，没有值的context对象, 用在需要context参数，但是又没有确切context可以选择的场景。

Background和TODO返回的context都使用相同的实现——emptyCtx。在context包中定义了两个全局变量：
```go
var (
	background = new(emptyCtx)
	todo       = new(emptyCtx)
)
```
每次调用Background都返回全局变量中的background； 同样每次调用TODO都返回全局变量中的todoemptyCtx是int类型的别名类型，且实现Context接口
的所有方法：
```go
type emptyCtx int

func (*emptyCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (*emptyCtx) Done() <-chan struct{} {
	return nil
}

func (*emptyCtx) Err() error {
	return nil
}

func (*emptyCtx) Value(key interface{}) interface{} {
	return nil
}
```
这里选择emptyCtx类型为int的别名，_是出于不同emptyCtx有不同的地址_。

### 四个派生context的方法
Background()、TODO()解决了context从无到有的问题。但是一个永不超时、永不过期、不带值的context对象对程序来说是没有作用的，能设置超时、设
置过期时间、携带值的context才是我们的需求。为了解决这些问题，context包提供了四个方法来派生满足需求的context：
```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc)
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc)
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)
func WithValue(parent Context, key, val interface{}) Context
```
四个方法有类似的结构： 接受一个父context对象，和一些别的定制参数；然后返回一个满足需求的context对象。

四个方法分为两组： 用于取消场景的context：WithCancel（通过调用返回的取消函数主动取消）， WithTimeout（超时取消）和WithDeadline（定时
取消）为一组；用于携带值的context：WithValue。

#### 取消context实现：
主动取消context的派生函数：WithCancel(parent Context)， 接收一个父context，返回新的子context对象和一个取消函数，调用cancel函数可以
让这个返回的子context取消，同时取消掉通过该子context上再次调用派生函数生产的所有后代context。

主动取消context派生函数的实现如下:
```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	c := newCancelCtx(parent)
	propagateCancel(parent, &c)
	return &c, func() { c.cancel(true, Canceled) }
}
```
WithCancel返回一个用于主动取消context的函数cancel。 CancelFunc定义如下：
```go
type CancelFunc func()
```

WithCancel有两个点值得研究：
- propagateCancel
- 基于cancelCtx实现的主动取消函数c.cancel(true, Canceled)

propagateCancel的作用是为传播取消机制做准备工作，传播取消机制是指当父context被取消的时候，所有的子context同步被取消。它的实现和接下来要
将的cancelCtx相关， 在讲解cancelCtx的最后会讲propagateCancel的实现。

cancelCtx的定义如下：
```go
type cancelCtx struct {
	Context

	mu       sync.Mutex            // protects following fields
	done     chan struct{}         // created lazily, closed by first cancel call
	children map[canceler]struct{} // set to nil by the first cancel call
	err      error                 // set to non-nil by the first cancel call
}
```
cancelCtx中内嵌了一个Context。在go语言中，通过类型嵌套，可以让嵌套类型拥有被嵌套类型的全部实现方法。context的实现用到了这个语言特性，目
的是让不同的Context接口实现专注于各种的实现，其他接口实现则委托给被嵌套者实现。

mu： 是一个同步变量，用于cancelCtx在多携程访问下的并发安全
done： 是一个信号通道，表明cancelCtx是否被关闭，为了让所有的子context同时对父context的关闭做出响应，这
里使用了通道而不是一个布尔类型。 Context接口中的Done方法在cancelCtx实现上就是返回该变量。
err： 表明当前cancelCtx被取消的原因。Context接口中的Err方法在cancelCtx实现上就是返回该变量。
children： 当前context派生出去的子context， 它是一个集合类型。 其中canceler的定义如下：
```go
type canceler interface {
	cancel(removeFromParent bool, err error)
	Done() <-chan struct{}
}
```

cancelCtx实现了Context接口和canceler接口，重点看看cancel的实现，它表明了取消时如何递归的通知子context取消：
```go
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
	if err == nil {          // 代码1
		panic("context: internal error: missing cancel error")
	}
	c.mu.Lock()
	if c.err != nil {       // 代码2
		c.mu.Unlock()
		return // already canceled
	}
	c.err = err
	if c.done == nil {
		c.done = closedchan     // 代码3
	} else {
		close(c.done)
	}
	for child := range c.children {     // 代码4
		// NOTE: acquiring the child's lock while holding parent's lock.
		child.cancel(false, err)
	}
	c.children = nil
	c.mu.Unlock()

	if removeFromParent {
		removeChild(c.Context, c)      // 代码5
	}
}
```
err： 说明是如何取消的，在context包的实现中，有两张取消方式： 主动取消， 超时取消。<br>
主动取消时err字段的值为Canceled：
```go
var Canceled = errors.New("context canceled")
```
定时取消时，err字段的值为DeadlineExceeded:
```go
var DeadlineExceeded error = deadlineExceededError{}

type deadlineExceededError struct{}

func (deadlineExceededError) Error() string   { return "context deadline exceeded" }

// ...other method implements of deadlineExceededError...
```
removeFromParent： 指明在取消cancelCtx是否把她从父context中移除掉。

代码1： 判断当前取消的错误类型，如果不是没有携带取消原因而取消会导致实现和Context接口约束不对应，没有取消原因时直接宕机。<br>
代码2： 判断是否已经被取消过了，如果取消过直接返回。<br>
代码3： 判断信号通道是否存在，如果存在则关闭，不存在赋值为一个预定义的关闭通道。之所以需要判断done通道是否为nil是因为cancelCtx的done通道
是延时初始化的。<br>
代码4: 遍历每一个子context并关闭它们，这也是关闭父context时会导致所有的子context关闭的实现。
代码5： 根据removeFromParent判断是否需要把当前context从父context中移除掉。

上面我们说的propagateCancel的实现和cancelCtx相关，cancelCtx的cancel方法最后会遍历每个子context被取消他们，cancelCtx的子context就是
通过propagateCancel和父context绑定的。
```go
func propagateCancel(parent Context, child canceler) {
	if parent.Done() == nil {                   // 代码1
		return // parent is never canceled
	}
	if p, ok := parentCancelCtx(parent); ok {   // 代码2
		p.mu.Lock()
		if p.err != nil {                      
			// parent has already been canceled
			child.cancel(false, p.err)
		} else {                               
			if p.children == nil {
				p.children = make(map[canceler]struct{})
			}
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {                                   // 代码3
		go func() {
			select {
			case <-parent.Done():
				child.cancel(false, parent.Err())
			case <-child.Done():
			}
		}()
	}
}
```
代码1： 当父context是不可取消的context时，这时不用把canceler加如到cancelCtx的children中。
代码2： 搜索父context是否存在cancelCtx， 如果存在且已被取消则直接取消到子context； 如果存在且没有取消，则把子context加入到父cancelCtx
的children里。
代码3： 如果没有父cancelCtx， 就需要通过一个携程去完成父context取消，子context同时取消的动作。因为有父cancelCtx时，子context是通过父
cancelCtx的cancel方法完成的。
通过propagateCancel后，父子context的取消动作得到了绑定。


取消context中的定时取消分为两种: 基于时间戳的定时context，和基于超时时间大小的超时取消。两者本质上没有区别，都是在某个时间触发context取
消。 定时取消给定了一个context取消的时间点，当时间到达给定的时间点，context触发取消； 超时取消给定一个时间大小，在当前时间的基础上经过指
定的时间大小后，context触发超时。 <br>
context包中超时取消的实现就是根据定时取消来实现的：
```go
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}
```
定时取消的实现是基于cancelCtx和Timer来完成的, 从上向下， 先来看WithDeadline的实现:
```go
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
	if cur, ok := parent.Deadline(); ok && cur.Before(d) {              // 代码1
		// The current deadline is already sooner than the new one.
		return WithCancel(parent)
	}
	c := &timerCtx{               // 代码2
		cancelCtx: newCancelCtx(parent),
		deadline:  d,
	}
	propagateCancel(parent, c)
	dur := time.Until(d)
	if dur <= 0 {               // 代码3
		c.cancel(true, DeadlineExceeded) // deadline has already passed
		return c, func() { c.cancel(true, Canceled) }
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {   // 代码4
		c.timer = time.AfterFunc(dur, func() {
			c.cancel(true, DeadlineExceeded)
		})
	}
	return c, func() { c.cancel(true, Canceled) }
}

```
代码1： 处理场景为传入的deadline时间比父context还要晚，这时直接返回父context的主动取消派生函数生产的context。
代码2： 新建一个timerCtx来完成定时取消，可以看到timerCtx实现和cancelCtx相关，后面会讲到。
代码3： 当传入的时间点是一个过去的时间时， context直接超时
代码4： 通过timer来的AfterFunc超时调用函数完成定时超时

下面来看看timerCtx定义和timerCtx的cancel实现：
```go
type timerCtx struct {
	cancelCtx
	timer *time.Timer // Under cancelCtx.mu.

	deadline time.Time
}

func (c *timerCtx) cancel(removeFromParent bool, err error) {
	c.cancelCtx.cancel(false, err)
	if removeFromParent {
		// Remove this timerCtx from its parent cancelCtx's children.
		removeChild(c.cancelCtx.Context, c)
	}
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}
```
timerCtx内嵌了cancelCtx， 其中timer时定时器，到deadline制定的时间后timer会触发timerCtx的cancel动作。<br>
timerCtx的cancel方法做三项事:
- 调用内嵌的cancelCtx取消context
- 是否从父context中remove掉当前contex
- 关闭timer


#### 携值context实现:
携带值的context实现如下：
```go
func WithValue(parent Context, key, val interface{}) Context {
	if key == nil {
		panic("nil key")
	}
	if !reflect.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &valueCtx{parent, key, val}
}

type valueCtx struct {
	Context
	key, val interface{}
}

func (c *valueCtx) Value(key interface{}) interface{} {
	if c.key == key {
		return c.val
	}
	return c.Context.Value(key)
}
```
类型valueCtx内嵌了一个Context，它的实现将Context接口中的非Value(key interface{})方法委托给内嵌的Context去完成，自己完成
Value(key interface{})接口实现。<br>
在调用Value时，如果当前valueCtx存储的key没有命中，则把Value操作传递给内嵌context。从实现中可以得出如下结论:
1. context携带的键值对中，键不能为nil且必须是可比较类型。
1. 每调用一次WithValue即会生产一个全新的context，新context指向父context。
1. context找值是从后往前的顺序查找的， 这里的从后往前是指WithValue(parent Context, key, val interface{})被调用的顺序。后面的
WithValue得到的context可以覆盖parent context中同key的value，但不是直接修改parent context中键值对的方式。
1. context上查找值时传入nil是安全的； 查找总是返回context链条上第一个key能匹配上的value，或者返回根context中的nil。

## 总结

