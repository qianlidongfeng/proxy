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

var(
	mytoken string = "333221abc"
	maxGetCount int = 1000
)
