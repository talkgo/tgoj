package main

import (
	"fmt"
	"log"
	"os"
	"tgoj/server/global"
	"tgoj/server/model"
)

func main() {
	mysqlConfig := global.CONFIG.Mysql
	fmt.Println(mysqlConfig)

	//err := global.DB.AutoMigrate(
	//	model.User{},
	//	model.Question{},
	//	)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	err := global.DB.Set(
		"gorm:table_options",
		"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4").
		AutoMigrate(
			&model.Question{},
			&model.User{},
			)
	if err != nil {
		log.Fatalln(err)
		os.Exit(0)
	}


	q1 := model.Question{
		Title: "两数之和",
		Description: "编写程序，实现两个整数相加的功能，并输出相加的结果。",
		Level: "简单",
		TestData: "2 10\n8 20\n-90 90\n0 0\n20 -30\n9999 1\n-10000 1\n",
		TestAnswer: "12\n28\n0\n0\n-10\n10000\n-9999\n",
		Example: "示例1\n    输入:1 2\n    输出:3",
		Tags: "入门级",
		MemoryLimit: 20<<20,
		TimeLimit: 1.0,
	}

	result := global.DB.Create(&q1)
	fmt.Println(*result)
}
