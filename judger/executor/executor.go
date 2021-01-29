package executor

import (
	"tgoj/judger"
	"tgoj/judger/verifier"
)

type Executor interface {
	// go 容器镜像，默认为 golang:1.15
	SetCompilerContainer(image string) error

	// 运行可执行文件的容器镜像，默认为 alpine:latest
	SetRunnerContainer(image string) error

	SetVerifier(v verifier.Verifier) error

	SetResultChan(resultCh chan<- judger.Result) error

	SetTaskChan(taskCh <-chan *judger.Task) error

	// 编译阶段的goroutine数量  如果设置了n>0 且 没有启动编译容器，会自动启动编译容器
	SetCompileConcurrency(n int) error

	// 执行阶段的goroutine数量
	SetRunConcurrency(n int) error

	// 校验阶段的goroutine数量
	SetVerifyConcurrency(n int) error

	// 启动编译容器
	EnableCompiler() error

	// 运行Executor
	Execute() error

	// 销毁Executor，立即销毁 或者 停止接收外部task 并 等待内部task执行完成
	Destroy(force bool) error
}
