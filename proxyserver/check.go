package proxyserver

import (
	"net/http"
	"strconv"
)

func checkRequest(r *http.Request,method string) bool{
	if r.Method != method{
		return false
	}
	r.ParseForm()
	if r.Form["count"] == nil{
		return false
	}
	if len(r.Form["count"])<1{
		return false
	}
	if r.Form["token"] == nil || r.Form["token"][0] !=mytoken{
		return false
	}
	count,err:= strconv.Atoi(r.Form["count"][0])
	if err != nil{
		return false
	}
	if count >maxGetCount||count<0{
		return false
	}
	return true
}
