package model

import "gorm.io/gorm"

type Question struct {
	gorm.Model
	Title       string  `json:"title" gorm:"type:varchar(100);unique;comment:题目名称"`
	Description string  `json:"description"  gorm:"comment:题目描述"`
	Level       string  `json:"level"  gorm:"type:varchar(10);comment:难度等级"`
	TestData    string  `json:"test_data"  gorm:"comment:评测输入的数据"`
	TestAnswer  string  `json:"test_answer"  gorm:"comment:评测的正确结果"`
	Example     string  `json:"example"  gorm:"comment:示例"`
	Tags        string  `json:"tags"  gorm:"comment:标签"`
	MemoryLimit int64   `json:"memory_limit"  gorm:"comment:内存限制"`
	TimeLimit   float64 `json:"time_limit"  gorm:"comment:时间限制"`
}
