# Flake

唯一编号生成器。

默认情况下，生成基于 Twitter SnowFlake 的 64 位唯一编号。

``` go
struct FlakeOption {
    Continuous bool     // 

}
```

指定起始值：

生成连续编号：
