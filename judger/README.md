# 评测机
## 使用前必看
- 需要设置环境变量`Resource`，值为存放`code、input、output、exe、answer`等资源的父目录，例如`mock`目录的路径。
- 先把编译和运行用的容器pull到本地，默认是golang:1.15 和 alpine:latest


## 结构
- executor: 将评测分为编译、运行、校验答案 三个阶段，支持CPU、内存、时间限制，但对容器的内存限制至少为6MB，实际建议限制内存最小值为16MB.
  - `EnableCompiler`会运行一个编译用的go容器，之后才能使用编译功能，所有编译工作都在该容器处理
  - 每次运行编译生成的可执行文件，都会启动一个专门运行该文件的容器，以实现环境隔离
  - 通过channel传递外部传入的评测任务、内部的编译、运行、校验任务
  - 每个阶段都支持并发，由多个goroutine监听channel
  - 支持同时运行多个goroutine执行compile、run、verify工作，具体使用参考`executor\docker_executor\dockerExecutor_test.go`的`TestDockerExecutor_Run`
  - 目前实现通过Docker执行每个阶段的工作，未来可以增加K8s或其他环境
  - 通过`context`实现多个goroutine的退出，每个goroutine监听的是同一个context变量
  - 调用`Destroy`销毁后，支持**立即销毁**和**等待内部任务处理完后再销毁（等待过程中停止接收外部传入的任务）**
- verifier: 比较标准答案和程序输出，保证这些文件都是相同编码，同样的换行(LF)
- errors: 评测相关的错误，包括编译、运行、校验等过程产生的问题


## 异常情况
- TLE: exited code: 143 SIGTERM terminated by timeout
- RE: exited code: 2
  - OOM
  - Index out of bound
- Killed: 
- 容器被删除:  exited code: 137
- 恶意系统调用: 
  - 删除文件: 以只读方式挂载可执行文件和输入目录，输出目录由于只挂载该用户的目录，即使删除（以及`/bin`等目录）也不会影响到其他人。
  - ...
- 容器启动失败:
  ```
  Error response from daemon: OCI runtime create failed: 
  container_linux.go:370:starting container process caused: 
  process_linux.go:459: container init caused: 
  process_linux.go:422: setting cgroup config for procHooks process caused: failed to write "16777216" to "/sys/fs/cgroup/memory/docker/ID/memory.memsw.limit_in_bytes": write /sys/fs/cgroup/memory/docker/ID/memory.memsw.limit_in_bytes: device or resource busy: unknown
  ```
  16777216，即 16MB，是 task中声明的8MB的memory限制的两倍，而写入文件是`memory.memsw.limit_in_bytes`，即包含swap的总内存，swap内存大小也是8MB，由于运行可执行文件很快，因此禁用swap。

  参考[该回答](https://unix.stackexchange.com/questions/412040/cgroups-memory-limit-write-error-device-or-resource-busy) ，可能是因为容器当时使用了超过我们设置的内存限制，导致容器只是处于created状态，而无法执行.
  
  解决方法: 容器内存限制最小设置为16MB.

### TODO
- 异常情况下的`Msg`还需要处理
- docker服务未启动下的错误处理: 触发`ErrUnknown`类型的错误
- Executor内部三种类型goroutine的动态扩缩容，可通过channel实现

## 缺点
- 需要运行所有测试用例，无法在某个用例出现问题时提前中止
- 每个题目只能有一个输入文件，因为运行可执行文件的容器只会执行一次命令，所以所有用例都存储在同一个文件中

## 运行
若以容器方式运行该应用，需要将宿主机的`/var/run/docker.sock`和`/usr/bin/docker`挂载到容器内相同路径。

