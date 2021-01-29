package judger

import "fmt"

type TaskStatus int

const (
	CREATED  TaskStatus = iota
	COMPILED            // 完成编译
	EXECUTED            // 执行结束
	FINISH              // 判题完成
)

type Task struct {
	ID         int64
	CodePath   string // 相对code 的路径
	AnswerPath string // 相对answer 的路径
	InputPath  string // 相对input 的路径
	OutputPath string // 相对output 的路径
	ExePath    string // 相对exe 的路径
	CpuPeriod  int64
	CpuQuota   int64
	Timeout    float64 // second
	Memory     int64   // in KB
	Status     TaskStatus
}

type Result struct {
	ID      int64
	Success bool
	//Message string // error when running executable, eg: OOM
	Error error // error when executing command
}

func (r Result) String() string {
	return fmt.Sprintf("ID: %v, Success: %v, Error: %v", r.ID, r.Success, r.Error)
}
