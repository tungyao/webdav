package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

var db *sql.DB

// StartDb 初始化数据库
func StartDb() {
	p := "main.db"
	if *isDocker == 1 {
		p = "./db/main.db"
	}
	dbs, err := sql.Open("sqlite3", p)
	if err != nil {
		log.Println(err)
	}
	db = dbs
}

type ShareObj struct {
	Path string
	Pass string
	Link string
	User string
	Idf  int
}

// GetShare 根据链接获取文件路径
func GetShare(shareName string) *ShareObj {
	row := db.QueryRow("select path,pass,idf from share where is_delete=0 and link=?", shareName)
	obj := &ShareObj{}
	err := row.Scan(&obj.Path, &obj.Pass, &obj.Idf)
	if err != nil {
		log.Println(err)
	}
	return obj
}
