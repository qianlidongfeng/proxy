package proxyserver

import (
	"encoding/json"
	"fmt"
	"github.com/qianlidongfeng/toolbox"
	"net/http"
	"strconv"
	"strings"
)

func HttpProxies(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET"{
		w.WriteHeader(400)
		return
	}
	r.ParseForm()
	if r.Form["count"] == nil{
		w.WriteHeader(400)
		return
	}
	count,err:= strconv.Atoi(r.Form["count"][0])
	if err != nil{
		w.WriteHeader(400)
		return
	}
	if count >1000||count<0{
		w.WriteHeader(400)
		return
	}
	rows,err:=db.Query(`select id,proxy from proxies where status = 1 order by distributetime asc limit ?`,count)
	if err != nil{
		log.Warn(err)
		w.Write([]byte(err.Error()))
		return
	}
	defer rows.Close()
	var proxy string
	var id int64
	var proxies []string
	var ids string
	for rows.Next(){
		rows.Scan(&id,&proxy)
		proxies=append(proxies,proxy)
		ids=ids+strconv.FormatInt(id,10)+","
	}
	rs,err:=json.Marshal(proxies)
	if err != nil{
		log.Warn(err)
		w.WriteHeader(500)
		return
	}
	now:=toolbox.GetTimeMilliStamp()
	ids=strings.TrimRight(ids,",")
	_,err=db.Exec(fmt.Sprintf(`update proxies set distributetime=? where id in (%s)`,ids),now)
	if err != nil{
		log.Warn(err)
		w.WriteHeader(500)
		return
	}
	w.Write([]byte(rs))
}