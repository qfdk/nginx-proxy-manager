package models

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"nginx-proxy-manager/app/config"
	"path"
)

var db *gorm.DB

func Init() {
	log.Println("[+] 初始化 SQLite ...")
	var err error
	dataDir := path.Join(config.GetAppConfig().InstallPath, "data.db")
	db, err = gorm.Open(sqlite.Open(dataDir), &gorm.Config{
		//Logger:      logger.Default.LogMode(logger.Info),
		PrepareStmt: true,
	})
	if err != nil {
		log.Println("[-] 初始化 SQLite 失败")
	}
	// Migrate the schema
	AutoMigrate(&Cert{})
	log.Println("[+] 初始化 SQLite 成功")
}

func AutoMigrate(model interface{}) {
	err := db.AutoMigrate(model)
	if err != nil {
		log.Println(err)
	}
}
func GetDbClient() *gorm.DB {
	return db
}
