package main

import (
	"log"
	"os"
)

func main() {
	// 会将output目录下的所有文件删除
	log.Println("rm output")
	err := os.RemoveAll("/output")
	if err != nil {
		log.Println(err)
	}

	// 只读方式挂载，不会删除
	log.Println("rm readonly input")
	err = os.RemoveAll("/input")
	if err != nil {
		log.Println(err)
	}
}
