package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username  string `json:"user_name" gorm:"comment:用户登录名"`
	Password  string `json:"-"  gorm:"comment:用户登录密码"`
	Email     string `json:"email"  gorm:"type:varchar(100);unique_index;comment:用户注册邮箱"`
	NickName  string `json:"nick_name" gorm:"type:varchar(100);default:系统用户;comment:用户昵称" `
	HeaderImg string `json:"header_img" gorm:"default:http://qmplusimg.henrongyi.top/head.png;comment:用户头像"`
}
