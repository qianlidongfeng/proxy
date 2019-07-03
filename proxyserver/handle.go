package proxyserver

import (
	"encoding/json"
	"fmt"
	"github.com/qianlidongfeng/toolbox"
	"net/http"
	"strconv"
	"strings"
)

func MyCollectionHttpProxies(w http.ResponseWriter, r *http.Request) {
	if !checkRequest(r,"GET"){
		w.WriteHeader(400)
		return
	}
	rows,err:=db.Query(`select id,proxy from proxies where status = 1 and source!="scan" and type="http" and total>1 order by distributetime asc limit ?`,r.Form["count"][0])
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
		proxies=append(proxies,"http://"+proxy)
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

func MyCollectionSock5Proxies(w http.ResponseWriter, r *http.Request) {
	if !checkRequest(r,"GET"){
		w.WriteHeader(400)
		return
	}
	rows,err:=db.Query(`select id,proxy from proxies where status = 1 and source!="scan" and type="sock5" and total>1 order by distributetime asc limit ?`,r.Form["count"][0])
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
		proxies=append(proxies,"socks5://"+proxy)
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

func MyScanHttpProxies(w http.ResponseWriter, r *http.Request) {
	if !checkRequest(r,"GET"){
		w.WriteHeader(400)
		return
	}
	rows,err:=db.Query(`select id,proxy from proxies where status = 1 and source="scan" and type="http" and total>1 order by distributetime asc limit ?`,r.Form["count"][0])
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
		proxies=append(proxies,"http://"+proxy)
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

func MyScanSock5Proxies(w http.ResponseWriter, r *http.Request) {
	if !checkRequest(r,"GET"){
		w.WriteHeader(400)
		return
	}
	rows,err:=db.Query(`select id,proxy from proxies where status = 1 and source="scan" and type="sock5" and total>1 order by distributetime asc limit ?`,r.Form["count"][0])
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
		proxies=append(proxies,"socks5://"+proxy)
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

func MyProxies(w http.ResponseWriter, r *http.Request){
	if !checkRequest(r,"GET"){
		w.WriteHeader(400)
		return
	}
	rows,err:=db.Query(`select id,type,proxy from proxies where status = 1 and source="scan" and total>1 order by distributetime asc limit ?`,r.Form["count"][0])
	if err != nil{
		log.Warn(err)
		w.Write([]byte(err.Error()))
		return
	}
	defer rows.Close()
	var proxy string
	var tp string
	var id int64
	var proxies []string
	var ids string
	for rows.Next(){
		rows.Scan(&id,&tp,&proxy)
		proxies=append(proxies,tp+"://"+proxy)
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