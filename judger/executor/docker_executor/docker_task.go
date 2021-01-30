package docker_executor

import (
	"sync"
	"tgoj/judger"
)

type compileTask struct {
	*judger.Task
}

// 可能有多个goroutine 同时写channel，在调用Destroy 非强制结束的时候，需要等到最后一个goroutine处理完之后才退出，因此加入wait group
// 调用Destroy时，当compile goroutine都结束之后，关闭runTask channel，因为对于这个channel，已经没用sender了。下面的runTaskChan也类似
type compileTaskChan struct {
	sync.WaitGroup
	ch chan compileTask
}

func newCompileTaskChan(size int) compileTaskChan {
	return compileTaskChan{
		ch: make(chan compileTask, size),
	}
}

type runTask struct {
	*judger.Task
	InputDirName   string
	InputFileName  string
	OutputDirName  string
	OutputFileName string
}

type runTaskChan struct {
	sync.WaitGroup
	ch chan runTask
}

func newRunTaskChan(size int) runTaskChan {
	return runTaskChan{
		ch: make(chan runTask, size),
	}
}

type verifyTask struct {
	*judger.Task
}

// 同runTaskChan
type verifyTaskChan struct {
	sync.WaitGroup
	ch chan verifyTask
}

func newVerifyTaskChan(size int) verifyTaskChan {
	return verifyTaskChan{
		ch: make(chan verifyTask, size),
	}
}
