package utils

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"os"
	"tgoj/server/config"
)

const (
	ConfigFile = "config.yaml"
)

func ReadYamlConfig()  (*config.Config, error) {
	cf := &config.Config{}
	dir, _ := os.Getwd()
	path := fmt.Sprintf("%s/server/%s", dir, ConfigFile)
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(yamlFile, cf); err != nil {
		return nil, err
	}
	return cf, nil

}


func StartMysql(m *config.Mysql) *gorm.DB {

	mysqlConfig := mysql.Config{
		DSN:                       m.Dsn(),   // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据版本自动配置
	}
	if db, err := gorm.Open(mysql.New(mysqlConfig),  &gorm.Config{}); err != nil {
		log.Fatalln("MySQL启动异常", err)
		os.Exit(0)
		return nil
	} else {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxIdleConns(m.MaxIdleConns)
		sqlDB.SetMaxOpenConns(m.MaxOpenConns)
		return db
	}
}
