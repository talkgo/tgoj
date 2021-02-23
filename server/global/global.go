package global

import (
	"gorm.io/gorm"
	"log"
	"os"
	"tgoj/server/config"
	"tgoj/server/utils"
)

var (
	CONFIG *config.Config
	DB *gorm.DB
)


func init()  {
	var err error
	CONFIG, err = utils.ReadYamlConfig()
	if err != nil {
		log.Fatalln("读取配置失败", err)
		os.Exit(0)
	}
	DB = utils.StartMysql(&CONFIG.Mysql)
}
