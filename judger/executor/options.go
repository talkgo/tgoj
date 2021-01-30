package executor

import (
	"tgoj/judger"
	"tgoj/judger/verifier"
)

type Option func(Executor) error

func EnableCompiler() Option {
	return func(executor Executor) error {
		return executor.EnableCompiler()
	}
}

func WithCompilerContainer(image string) Option {
	return func(executor Executor) error {
		return executor.SetCompilerContainer(image)
	}
}

func WithRunnerContainer(image string) Option {
	return func(executor Executor) error {
		return executor.SetRunnerContainer(image)
	}
}

func WithVerifier(v verifier.Verifier) Option {
	return func(executor Executor) error {
		return executor.SetVerifier(v)
	}
}

func WithTaskChan(taskCh <-chan *judger.Task) Option {
	return func(executor Executor) error {
		return executor.SetTaskChan(taskCh)
	}
}

func WithResultChan(resultCh chan<- judger.Result) Option {
	return func(executor Executor) error {
		return executor.SetResultChan(resultCh)
	}
}

func WithCompileConcurrency(n int) Option {
	return func(executor Executor) error {
		return executor.SetCompileConcurrency(n)
	}
}

func WithRunConcurrency(n int) Option {
	return func(executor Executor) error {
		return executor.SetRunConcurrency(n)
	}
}

func WithVerifyConcurrency(n int) Option {
	return func(executor Executor) error {
		return executor.SetVerifyConcurrency(n)
	}
}
