package docker_executor

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"tgoj/judger"
	"tgoj/judger/errors"
	"tgoj/judger/executor"
	"tgoj/judger/utils"
	"tgoj/judger/verifier"
)

const (
	DefaultCompileContainerName = "golang:1.15"
	DefaultRunnerContainerName  = "alpine:latest"
	//DEBUG = true
	DefaultChannelSize = 100
)

var ResourcePath string

func init() {
	// 存放评测相关文件的目录，例如mock目录
	ResourcePath = os.Getenv("Resource")
}

type Status int

const (
	CREATED Status = iota
	RUNNING
	DESTROYING // 等待所有任务结束
	DESTROYED  // 已销毁，不能使用
)

var _ executor.Executor = (*DockerExecutor)(nil)

type DockerExecutor struct {
	sync.Mutex
	ctx        context.Context
	cancelFunc context.CancelFunc

	resultCh chan<- judger.Result
	taskCh   <-chan *judger.Task

	compileTaskCh compileTaskChan
	runTaskCh     runTaskChan
	verifyTaskCh  verifyTaskChan

	cli                    *client.Client // docker client
	compilerContainerImage string
	compilerContainerID    string
	runnerContainerImage   string
	enableCompile          bool
	verifier               verifier.Verifier
	status                 Status
}

/****  Initialization      *****/
func (d *DockerExecutor) SetVerifier(v verifier.Verifier) error {
	d.verifier = v
	return nil
}

// 如果设置了n>0 且 没有启动编译容器，会自动启动编译容器
func (d *DockerExecutor) SetCompileConcurrency(n int) error {
	if n <= 0 {
		return fmt.Errorf("if set, compile concurrency must be greater than 0, but received %v", n)
	}

	if d.compilerContainerID == "" {
		if err := d.startCompiler(); err != nil {
			return err
		}
	}

	for i := 0; i < n; i++ {
		d.compileTaskCh.Add(1)
		go d.Compile()
	}
	return nil
}

func (d *DockerExecutor) SetRunConcurrency(n int) error {
	if n <= 0 {
		return fmt.Errorf("if set, run concurrency must be greater than 0, but received %v", n)
	}

	for i := 0; i < n; i++ {
		d.runTaskCh.Add(1)
		go d.Run()
	}
	return nil
}

func (d *DockerExecutor) SetVerifyConcurrency(n int) error {
	if n <= 0 {
		return fmt.Errorf("if set, verify concurrency must be greater than 0, but received %v", n)
	}

	for i := 0; i < n; i++ {
		d.verifyTaskCh.Add(1)
		go d.Verify()
	}
	return nil
}

func (d *DockerExecutor) SetCompilerContainer(image string) error {
	d.compilerContainerImage = image
	return nil
}

func (d *DockerExecutor) SetRunnerContainer(image string) error {
	d.runnerContainerImage = image
	return nil
}

func (d *DockerExecutor) SetResultChan(resultCh chan<- judger.Result) error {
	d.resultCh = resultCh
	return nil
}

func (d *DockerExecutor) SetTaskChan(taskCh <-chan *judger.Task) error {
	d.taskCh = taskCh
	return nil
}

func (d *DockerExecutor) EnableCompiler() error {
	if d.compilerContainerID == "" {
		return d.startCompiler()
	}
	return nil
}

func New(opts ...executor.Option) *DockerExecutor {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	d := &DockerExecutor{
		ctx:                    ctx,
		cancelFunc:             cancelFunc,
		cli:                    cli,
		compilerContainerImage: DefaultCompileContainerName,
		runnerContainerImage:   DefaultRunnerContainerName,
		runTaskCh:              newRunTaskChan(DefaultChannelSize),
		compileTaskCh:          newCompileTaskChan(DefaultChannelSize),
		verifyTaskCh:           newVerifyTaskChan(DefaultChannelSize),
		verifier:               verifier.StandardVerifier{},
		status:                 CREATED,
	}

	for _, opt := range opts {
		if err = opt(d); err != nil {
			log.Fatal(err)
		}
	}
	return d
}

/****  Operation      *****/
func (d *DockerExecutor) Destroy(force bool) error {
	// 停止所有 goroutine
	// 如果在RUNNING状态收到退出的信息，说明是强制退出，不会处理内部还有的任务
	// 如果在DESTROYING状态收到退出的信息，则是非强制退出，可以依次等待每个阶段残留的任务运行完成后再退出
	//     每个阶段处理完task后，关闭发往下一个阶段的channel
	if !force {
		d.status = DESTROYING
	}
	d.cancelFunc()

	d.compileTaskCh.Wait()
	close(d.runTaskCh.ch)

	d.runTaskCh.Wait()
	close(d.verifyTaskCh.ch)

	d.verifyTaskCh.Wait()
	d.status = DESTROYED

	// 删除容器
	log.Println("remove compile container")
	if d.compilerContainerID != "" {
		if err := d.cli.ContainerRemove(context.Background(), d.compilerContainerID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

func (d *DockerExecutor) Execute() error {
	d.status = RUNNING
	for {
		select {
		case <-d.ctx.Done():
			// compileTaskCh 只有一个sender，所以可以直接关闭
			close(d.compileTaskCh.ch)
			return nil
		case task := <-d.taskCh: // 接收外部传入的任务，并根据任务状态执行
			//log.Println("execute task: ", task.ID)
			switch task.Status {
			case judger.CREATED:
				d.compileTaskCh.ch <- compileTask{
					Task: task,
				}
			case judger.COMPILED:
				inputDir, inputFile := filepath.Split(task.InputPath)
				outputDir, outputFile := filepath.Split(task.OutputPath)
				d.runTaskCh.ch <- runTask{
					Task:           task,
					InputDirName:   inputDir,
					InputFileName:  inputFile,
					OutputDirName:  outputDir,
					OutputFileName: outputFile,
				}
			case judger.EXECUTED:
				d.verifyTaskCh.ch <- verifyTask{Task: task}
			}
		}
	}
}

func (d *DockerExecutor) Compile() {
	defer func() {
		d.compileTaskCh.Done()
	}()

	var task compileTask
	var ok bool
	for {
		select {
		case <-d.ctx.Done():
			d.finishCompile()
			return
		case task, ok = <-d.compileTaskCh.ch:
			if !ok {
				d.finishCompile()
				return
			}
			//log.Println("compile task: ", task.ID)
			d.processCompileTask(task)
		}
	}
}

func (d *DockerExecutor) finishCompile() {
	if d.status == DESTROYING { // 非强制退出
		log.Println("processing left compile task")
		// compileTaskCh 已关闭，因为带缓冲，处理完channel内剩余task再退出
		for task := range d.compileTaskCh.ch {
			d.processCompileTask(task)
		}
	}
}

func (d *DockerExecutor) processCompileTask(task compileTask) {
	// 可执行文件相对exe目录的路径 与 源代码文件相对code目录的路径 相同
	task.ExePath = strings.TrimSuffix(task.CodePath, ".go")
	err, rerun := d.compile(task)

	if err != nil {
		d.resultCh <- judger.Result{
			ID:      task.ID,
			Success: false,
			Error:   err,
		}
		return
	}

	if rerun {
		return
	}

	inputDir, inputFile := filepath.Split(task.InputPath)
	outputDir, outputFile := filepath.Split(task.OutputPath)
	d.runTaskCh.ch <- runTask{
		Task:           task.Task,
		InputDirName:   inputDir,
		InputFileName:  inputFile,
		OutputDirName:  outputDir,
		OutputFileName: outputFile,
	}
}

func (d *DockerExecutor) compile(task compileTask) (err error, rerun bool) {
	defer func() {
		if rerun, err = d.checkCompilerError(err); err != nil {
			return
		} else if rerun {
			d.compileTaskCh.ch <- task
		}
	}()

	input, output := task.CodePath, task.ExePath
	// 保证目录存在
	outputDir := filepath.Dir(output)
	if outputDir != "." {
		utils.CheckDirectoryExist(fmt.Sprintf("%s/exe/%s", ResourcePath, outputDir))
	}

	resp, err := d.cli.ContainerExecCreate(context.Background(), d.compilerContainerID, types.ExecConfig{
		// disable optimize and inline   -gcflags '-N -l'
		Cmd: []string{"sh", "-c",
			fmt.Sprintf("go build -o /exe/%s /code/%s", output, input)},
		AttachStderr: true,
		AttachStdout: true,
	})
	if err != nil {
		return
	}

	response, err := d.cli.ContainerExecAttach(context.Background(), resp.ID, types.ExecStartCheck{})
	if err != nil {
		return
	}
	defer response.Close()

	err = d.cli.ContainerExecStart(context.Background(), resp.ID, types.ExecStartCheck{})
	if err != nil {
		return
	}

	commandOutput, err := utils.ReadFromBIO(response.Reader)
	if err != nil {
		return
	}

	inspect, err := d.cli.ContainerExecInspect(context.Background(), resp.ID)
	if err != nil {
		return
	}
	if inspect.ExitCode != 0 {
		return errors.New(errors.CE, commandOutput), false
	}

	return
}

func (d *DockerExecutor) Run() {
	defer func() {
		d.runTaskCh.Done()
	}()

	var task runTask
	var ok bool
	for {
		select {
		case <-d.ctx.Done():
			d.finishRun()
			return
		case task, ok = <-d.runTaskCh.ch:
			if !ok {
				d.finishRun()
				return
			}
			//log.Println("run task: ", task.ID)
			d.processRunTask(task)
		}
	}
}

func (d *DockerExecutor) finishRun() {
	if d.status == DESTROYING { // 非强制退出
		log.Println("processing left run task")
		// runTaskCh 已关闭，因为带缓冲，处理完channel内剩余task再退出
		for task := range d.runTaskCh.ch {
			d.processRunTask(task)
		}
	}
}

func (d *DockerExecutor) processRunTask(task runTask) {
	err := d.run(task)
	//log.Println("run task finish: ", task.ID, err)
	if err != nil {
		d.resultCh <- judger.Result{
			ID:      task.ID,
			Success: false,
			Error:   err,
		}
		return
	}

	task.Task.Status = judger.EXECUTED
	d.verifyTaskCh.ch <- verifyTask{Task: task.Task}
}

func (d *DockerExecutor) run(task runTask) error {
	// 保证目录存在
	if task.OutputDirName != "." {
		utils.CheckDirectoryExist(fmt.Sprintf("%s/output/%s", ResourcePath, task.OutputDirName))
	}

	resp, err := d.cli.ContainerCreate(context.Background(), &container.Config{
		// echo $(tr "\n" " " < /input/1.go) | timeout 2.5 /exe > /output/1.txt
		Cmd: []string{"sh", "-c",
			fmt.Sprintf("echo $(tr \"\\n\" \" \" < /input/%s) | timeout %v /exe > /output/%s",
				task.InputFileName, strconv.FormatFloat(task.Timeout, 'f', 4, 32), task.OutputFileName),
		},
		//Cmd: []string{"sh", "-c", "while true; do sleep 100; done"}, // for debug
		Image:        d.runnerContainerImage,
		AttachStdout: true,
		AttachStderr: true,
	}, &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s/exe/%s:/exe:ro", ResourcePath, task.ExePath),
			fmt.Sprintf("%s/output/%s:/output", ResourcePath, task.OutputDirName),
			fmt.Sprintf("%s/input/%s:/input:ro", ResourcePath, task.InputDirName),
		},
		AutoRemove: true,
		Resources: container.Resources{
			Memory:     task.Memory,
			MemorySwap: task.Memory,
			CPUPeriod:  task.CpuPeriod,
			CPUQuota:   task.CpuQuota,
		},
	}, nil, nil, "")
	if err != nil {
		log.Println(task.ID, err)
		return err
	}

	hijackedResponse, err := d.cli.ContainerAttach(context.Background(), resp.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		log.Println(task.ID, err)
	}

	if err = d.cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Println(task.ID, err)
		return err
	}

	status, err := d.exec(resp.ID)
	if err != nil {
		log.Println(task.ID, err)
		return err
	}

	if status.Error != nil && len(status.Error.Message) > 0 {
		err = errors.New(errors.ENV, status.Error.Message)
	} else {
		var msg string
		msg, err = utils.ReadFromBIO(hijackedResponse.Reader)
		if err != nil {
			return err
		}

		if status.StatusCode == 0 {
			return nil
		}

		if v, ok := errors.ExitedCode2JudgerError[status.StatusCode]; ok {
			err = errors.New(v, msg)
		} else {
			err = errors.New(errors.UNKNOWN, msg)
		}
	}
	//inspect, err := d.cli.ContainerInspect(context.Background(), resp.ID)
	//log.Println(inspect)
	return err
}

func (d *DockerExecutor) Verify() {
	defer func() {
		d.verifyTaskCh.Done()
	}()

	var task verifyTask
	var ok bool
	for {
		select {
		case <-d.ctx.Done():
			d.finishVerify()
			return
		case task, ok = <-d.verifyTaskCh.ch:
			if !ok {
				d.finishVerify()
				return
			}
			//log.Println("verify task: ", task.ID)
			d.processVerifyTask(task)
		}
	}
}

func (d *DockerExecutor) finishVerify() {
	if d.status == DESTROYING { // 非强制退出
		log.Println("processing left verify task")
		// verifyTaskCh 已关闭，因为带缓冲，处理完channel内剩余task再退出
		for task := range d.verifyTaskCh.ch {
			d.processVerifyTask(task)
		}
	}
}

func (d *DockerExecutor) processVerifyTask(task verifyTask) {
	_, err := d.verifier.Verify(fmt.Sprintf("%s/output/%s", ResourcePath, task.OutputPath),
		fmt.Sprintf("%s/answer/%s", ResourcePath, task.AnswerPath))

	d.resultCh <- judger.Result{
		ID:      task.ID,
		Success: err == nil,
		Error:   err,
	}
}

func (d *DockerExecutor) exec(id string) (container.ContainerWaitOKBody, error) {
	statusCh, errCh := d.cli.ContainerWait(context.Background(), id, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		return container.ContainerWaitOKBody{}, err
	case status := <-statusCh:
		return status, nil
	}
}

// 启动一个编译容器 并记录容器ID
func (d *DockerExecutor) startCompiler() error {
	resp, err := d.cli.ContainerCreate(context.Background(), &container.Config{
		Tty:       true,
		OpenStdin: true,
		Image:     d.compilerContainerImage,
	}, &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s/code:/code", ResourcePath),
			fmt.Sprintf("%s/exe:/exe", ResourcePath),
		},
	}, nil, nil, "")
	if err != nil {
		return err
	}

	d.compilerContainerID = resp.ID
	return d.cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
}

func (d *DockerExecutor) restartCompiler() error {
	d.Lock()
	defer d.Unlock()

	_, err := d.cli.ContainerInspect(context.Background(), d.compilerContainerID)
	if errdefs.IsNotFound(err) {
		return d.startCompiler()
	}
	return err
}

// if recover from error by restarting compiler, and restart success, then need to rerun
// if fail to restart compiler, don't rerun
func (d *DockerExecutor) checkCompilerError(err error) (rerun bool, e error) {
	if err == nil || errors.IsError(err, errors.CE) {
		return false, err
	}

	log.Println("compiler error: ", err)
	if err = d.restartCompiler(); err != nil {
		return false, err
	}
	return true, nil
}
