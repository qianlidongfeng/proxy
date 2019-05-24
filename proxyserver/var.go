package proxyserver

import (
	"database/sql"
	"github.com/qianlidongfeng/loger"
)

var (
	log loger.Loger
	db *sql.DB
	proxieTable string
)
