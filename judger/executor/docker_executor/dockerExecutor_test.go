package docker_executor

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"tgoj/judger"
	"tgoj/judger/executor"
	"tgoj/judger/verifier"
	"time"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func TestDockerExecutor_Run(t *testing.T) {
	// 在该Test中，resultCh必须有缓冲，否则Executor在非强制Destroy时会死锁
	taskCh, resultCh := make(chan *judger.Task), make(chan judger.Result, 100)
	var options = []executor.Option{
		executor.EnableCompiler(),
		executor.WithResultChan(resultCh),  // 必须项
		executor.WithTaskChan(taskCh),      // 必须项
		executor.WithCompileConcurrency(3), // 必须项
		executor.WithRunConcurrency(3),     // 必须项
		executor.WithVerifyConcurrency(3),  // 必须项
		executor.WithVerifier(verifier.StandardVerifier{}),
	}
	dockerExecutor := New(options...)

	// 用于等待Executor Destroy结束后再退出主协程
	ch := make(chan struct{})

	go func() {
		var n = 6
		var tasks []judger.Task
		for i := 0; i < 7; i++ {
			tasks = append(tasks, judger.Task{
				ID:         int64(i),
				AnswerPath: "1.txt",
				InputPath:  "1.txt",
				OutputPath: fmt.Sprintf("1//%v.txt", i),
				CpuPeriod:  100000,
				CpuQuota:   50000,
				Timeout:    1.0,
				Memory:     20 << 20, // 20 MB for WSL, 10MB for linux like ubuntu、centos
				Status:     judger.CREATED,
			})
		}

		// 由于judger运行在windows环境，docker 运行在wsl中，使用filepath.Join 生成的path不适用于linux环境
		tasks[0].CodePath = "1//success.go" // filepath.Join("1", "success.go")
		tasks[1].CodePath = "out_of_bound.go"
		tasks[2].CodePath = "oom.go"
		tasks[3].CodePath = "ce.go"
		tasks[4].CodePath = "timeout.go"
		tasks[5].CodePath = "success.go"
		tasks[6].CodePath = "rm.go"

		for i := 0; i < n; i++ {
			log.Println("put task: ", i)
			taskCh <- &tasks[i]
		}

		time.Sleep(time.Second * 2)
		log.Println("destroy start")
		if err := dockerExecutor.Destroy(false); err != nil {
			log.Println(err)
		}

		for i := 0; i < n; i++ {
			res := <-resultCh
			log.Println(res)
		}

		//if err := dockerExecutor.Destroy(false); err != nil {
		//	log.Println(err)
		//}
		ch <- struct{}{}
	}()

	log.Println("start executing")
	if err := dockerExecutor.Execute(); err != nil {
		log.Println(err)
	}
	log.Println("stop execute")
	<-ch
}

func TestDockerExecutor_Exec(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
	}
}

func TestDockerExecutor_PullImage(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}

	reader, err := cli.ImagePull(context.Background(), "golang:1.15", types.ImagePullOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)
}

func TestDockerExecutor_Compile(t *testing.T) {
	dockerExecutor := New(executor.EnableCompiler())
	task := compileTask{
		Task: &judger.Task{
			ID:         1,
			AnswerPath: "1.txt",
			InputPath:  "1.txt",
			OutputPath: fmt.Sprintf("1//%v.txt", 1),
			CpuPeriod:  100000,
			CpuQuota:   50000,
			Timeout:    1.0,
			Memory:     8 << 20, // 16 MB
			Status:     judger.CREATED,
			CodePath:   "1//success.go",
			ExePath:    "1//success",
		},
	}
	err, _ := dockerExecutor.compile(task)
	if err != nil {
		log.Fatal(err)
	}
}

func TestDockerExecutor_RunVerifier(t *testing.T) {
	var v verifier.StandardVerifier
	var outputs = []string{
		fmt.Sprintf("%v\\mock\\output\\success.txt", ResourcePath),
		fmt.Sprintf("%v\\mock\\output\\oob.txt", ResourcePath), // empty
		fmt.Sprintf("%v\\mock\\output\\1.txt", ResourcePath),   // output more
		fmt.Sprintf("%v\\mock\\output\\2.txt", ResourcePath),   // output less
	}

	for _, output := range outputs {
		cases, err := v.Verify(output,
			fmt.Sprintf("%v\\mock\\standard_output\\1.txt", ResourcePath))
		log.Println(fmt.Sprintf("%v pass %v cases with err: %v", output, cases, err))
	}
}

func TestSprintf(t *testing.T) {
	var output = "output"
	var input = "input"
	log.Println(fmt.Sprintf("build -o /exe/%s /code/%s", output, input))

	strs := []string{"sh", "-c",
		fmt.Sprintf("\" timeout %v /exe < /input/%s > /output/%s \"",
			strconv.FormatFloat(2.5, 'f', 4, 32), "1.go", "1.txt")}
	fmt.Println(strings.Join(strs, " "))
}
