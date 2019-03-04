# 函数记忆问题
[函数记忆问题](https://en.wikipedia.org/wiki/Memoization): 缓存函数结果，达到多次调用但只须计算一次结果。
即连续对同一个函数的调用，如何保证只要一次调用被触发，其它访问返回缓存结果。

# go解决函数记忆问题的两种方法
1. 通过同步锁实现： 见mutexmemo.go
2. 通过监控goroutine实现: 见monitormemo.go