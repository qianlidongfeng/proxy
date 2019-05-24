package checker

import (
	"database/sql"
	"fmt"
	"github.com/qianlidongfeng/httpclient"
	"github.com/qianlidongfeng/loger"
	"github.com/qianlidongfeng/loger/netloger"
	"github.com/qianlidongfeng/proxy"
	"github.com/qianlidongfeng/proxy/clientserver"
	"github.com/qianlidongfeng/toolbox"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

type infoIp138 struct{
	ip string
	city string
}

type ProxyChecker struct{
	db          *sql.DB
	cfg         Config
	loger       loger.Loger
	stdout      *os.File
	rpcConns     []*grpc.ClientConn
	rpcClients   []clientserver.HttpClient
	proxyPool   chan Feilds
	stmtGet     *sql.Stmt
	stmtFail    *sql.Stmt
	stmtSuccess *sql.Stmt
	url         string
	success     int
	selfIp      string
	selfClient  httpclient.HttpClient
	clientIndex int
	mu sync.Mutex
	wg sync.WaitGroup
}

func NewProxyChecker() ProxyChecker{
	return ProxyChecker{}
}

func (this *ProxyChecker) Init(configPath string) error{
	err:=toolbox.LoadConfig(configPath,&this.cfg)
	if err != nil{
		return err
	}
	if this.cfg.Debug == false{
		appPath,err:=toolbox.AppPath()
		if err != nil{
			log.Fatal(err)
		}
		this.stdout, err = os.OpenFile(appPath+".log", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0644)
		if err != nil{
			log.Fatal(err)
		}
		log.SetOutput(this.stdout)
	}
	if this.cfg.Loger.LogType == "netlog" && this.cfg.Debug==false{
		lg:=netloger.NewSqloger()
		err=lg.Init(this.cfg.Loger.DB)
		if err != nil{
			return err
		}
		this.loger=lg
	}else{
		this.loger=loger.NewLocalLoger()
	}
	this.db,err= toolbox.InitMysql(this.cfg.DB)
	if err != nil{
		this.loger.Fatal(err)
	}
	fields:=make(map[string]string)
	fields["id"]="bigint(20) unsigned"
	fields["proxy"]="varchar(32)"
	fields["area"]="varchar(64)"
	fields["type"]="varchar(16)" //代理类型
	fields["category"]="tinyint(4) unsigned" //匿名度
	fields["status"]="tinyint(4) unsigned" //代理状态
	fields["success"]="int(11) unsigned"  //成功总次数
	fields["total"]="int(11) unsigned"//检测总次数
	fields["successrate"]="tinyint(4) unsigned"//成功率
	fields["exist"]="int(11) unsigned" //存在总时长
	fields["lastcheck"]="int(11) unsigned"
	fields["distributetime"]="bigint(20) unsigned"
	fields["source"]="varchar(16)" //代理来源
	fields["ctime"]="datetime" //创建时间
	err=toolbox.CheckTable(this.db,this.cfg.DB.Table,fields)
	if err != nil{
		this.loger.Fatal(err)
	}
	this.stmtGet,err=this.db.Prepare(fmt.Sprintf(`SELECT id,proxy,type,success,total,ctime FROM %s`,this.cfg.DB.Table))
	if err != nil{
		this.loger.Fatal(err)
	}
	this.stmtSuccess,err=this.db.Prepare(fmt.Sprintf(`UPDATE %s SET success=?,successrate=?,total=?,exist=?,status=?,lastcheck=? WHERE id=?`,this.cfg.DB.Table))
	if err != nil{
		this.loger.Fatal(err)
	}
	this.stmtFail,err=this.db.Prepare(fmt.Sprintf(`UPDATE %s SET successrate=?,total=?,exist=?,status=?,lastcheck=? WHERE id=?`,this.cfg.DB.Table))
	if err != nil{
		this.loger.Fatal(err)
	}
	if len(this.cfg.RpcClientServer)==0{
		this.loger.Fatal("rpcclient server is empty")
	}
	for _,v:=range this.cfg.RpcClientServer{
		rpcConn, err := grpc.Dial(
			v,
			grpc.WithInsecure(),
		)
		if err != nil{
			this.loger.Fatal(err)
		}
		if !this.checkRpcReady(rpcConn,2*time.Second){
			this.loger.Fatal(fmt.Sprintf("bad rpcclient server:%s",v))
		}
		rpcClient:=clientserver.NewHttpClient(rpcConn)
		this.rpcConns=append(this.rpcConns,rpcConn)
		this.rpcClients=append(this.rpcClients,rpcClient)
	}
	this.proxyPool = make(chan Feilds,this.cfg.Thread)
	this.url="http://2019.ip138.com/ic.asp"
	this.selfClient= httpclient.NewHttpClient()
	this.selfClient.SetTimeOut(10*time.Second)
	return nil
}

func (this *ProxyChecker) GetRpcClient() clientserver.HttpClient{
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.clientIndex >= len(this.rpcClients){
		this.clientIndex=0
	}
	client:=this.rpcClients[this.clientIndex]
	this.clientIndex++
	return client
}

func (this *ProxyChecker) Check() error{
	this.selfIp=this.getSelfIp()
	for i:=0;i<this.cfg.Thread;i++{
		go func(index int){
			for{
				this.Do()
			}
		}(i)
	}
	for{
		this.success=0
		rows,err:=this.stmtGet.Query()
		if err != nil{
			this.loger.Fatal(err)
		}
		for rows.Next(){
			fields := Feilds{}
			err=rows.Scan(&fields.id,&fields.proxy,&fields.tp,&fields.success,&fields.total,&fields.ctime)
			if err != nil{
				this.loger.Fatal(err)
			}
			this.wg.Add(1)
			this.proxyPool<-fields
		}
		this.wg.Wait()
		err=this.DeleteInvalidProxy()
		if err != nil{
			rows.Close()
			this.loger.Fatal(err)
		}
		this.selfIp=this.getSelfIp()
		rows.Close()
		this.loger.Msg("totoal success",this.success)
		time.Sleep(time.Second)
	}
	return nil
}

func (this *ProxyChecker)Do(){
	var fields Feilds
	fields=<-this.proxyPool
	defer this.wg.Done()
	var err error
	var resp *clientserver.Respone
	if strings.ToLower(fields.tp)=="http" || strings.ToLower(fields.tp)=="https"{
		resp,err=this.GetRpcClient().Get(context.Background(),&clientserver.ProxyInfo{Proxy:"http://"+fields.proxy,Type:"http"})
	}else if strings.ToLower(fields.tp)=="sock5"{
		resp,err=this.GetRpcClient().Get(context.Background(),&clientserver.ProxyInfo{Proxy:fields.proxy,Type:"sock5"})
	}
	if err != nil || resp == nil{
		code:=status.Code(err)
		if code != codes.Unknown && code !=codes.OK{
			this.loger.Warn(fmt.Sprintf("error code:%d, message:%s",code,err.Error()))
			time.Sleep(time.Second*60)
			return
		}
		this.onFail(fields)
		return
	}
	html,err:=toolbox.GbkToUtf8(string(resp.Content))
	if err != nil{
		this.onFail(fields)
		return
	}
	this.Parse(html,fields)
}

type Feilds struct{
	id int64
	proxy string
	tp string
	success int
	total int
	ctime string
}

func (this *ProxyChecker) Parse(html string,fields Feilds){
	ip,err := this.getIp(html)
	if err != nil{
		this.onFail(fields)
		return
	}
	if ip==this.selfIp{
		this.onFail(fields)
	}else{
		this.onSuccess(fields)
	}
}

func (this *ProxyChecker) Realse(){
	for _,v := range this.rpcConns{
		v.Close()
	}
	this.stmtGet.Close()
	this.stmtFail.Close()
	this.stmtSuccess.Close()
	this.db.Close()
	this.stdout.Close()
	this.loger.Close()
}

func (this *ProxyChecker) checkRpcReady(conn *grpc.ClientConn,timeout time.Duration) bool{
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var state connectivity.State
	for state = conn.GetState(); state != connectivity.Ready && conn.WaitForStateChange(ctx, state); state = conn.GetState() {
	}
	return state == connectivity.Ready
}

func (this *ProxyChecker) onSuccess(fields Feilds){
	fields.success++
	fields.total++
	successrate:=float64(fields.success)/float64(fields.total)
	ctime,err:=time.Parse("2006-01-02 15:04:05", fields.ctime)
	if err != nil{
		this.loger.Fatal(err)
	}
	now,err:=time.Parse("2006-01-02 15:04:05",time.Now().Format("2006-01-02 15:04:05"))
	if err != nil{
		this.loger.Fatal(err)
	}
	_,err=this.stmtSuccess.Exec(fields.success,math.Trunc(successrate*1e2) * 1e-2*100,fields.total,now.Unix()-ctime.Unix(),1,now.Unix(),fields.id)
	if err != nil{
		this.loger.Fatal(err)
	}
	this.success++
}

func (this *ProxyChecker) onFail(fields Feilds){
	fields.total++
	successrate:=float64(fields.success)/float64(fields.total)
	ctime,err:=time.Parse("2006-01-02 15:04:05", fields.ctime)
	if err != nil{
		this.loger.Fatal(err)
	}
	now,err:=time.Parse("2006-01-02 15:04:05",time.Now().Format("2006-01-02 15:04:05"))
	if err != nil{
		this.loger.Fatal(err)
	}
	_,err=this.stmtFail.Exec(math.Trunc(successrate*1e2) * 1e-2*100,fields.total,now.Unix()-ctime.Unix(),0,now.Unix(),fields.id)
	if err != nil{
		this.loger.Fatal(err)
	}
}


func (this *ProxyChecker) getSelfIp() string{
	for{
		r,err:= this.selfClient.Get(this.url)
		if err != nil{
			this.loger.Msg("selfip",err)
			time.Sleep(10*time.Second)
			continue
		}
		html,err := toolbox.GbkToUtf8(r.Html)
		if err != nil{
			this.loger.Msg("selfip",err)
			time.Sleep(10*time.Second)
			continue
		}
		info,err := proxy.GetIpFrom138(html)
		if err != nil{
			this.loger.Msg("selfip",err)
			time.Sleep(10*time.Second)
			continue
		}
		return info.IP
	}
}

func (this *ProxyChecker) getIp(html string) (string,error){
	info,err:=proxy.GetIpFrom138(html)
	return info.IP,err
}

func (this *ProxyChecker) DeleteInvalidProxy() error{
	_,err:=this.db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE successrate<? AND exist<?`,this.cfg.DB.Table),this.cfg.BadSuccessrate,this.cfg.BadExist)
	if err != nil{
		this.loger.Fatal(err)
	}
	return nil
}

func (this *ProxyChecker) ResetID() error{
	conn,err:=this.db.Begin()
	if err != nil{
		this.loger.Fatal(err)
	}
	_,err=conn.Exec(`alter table test drop column id`)
	if err != nil{
		conn.Rollback()
		this.loger.Fatal(err)
	}
	_,err=conn.Exec(`alter table test add id bigint(20) unsigned not null primary key auto_increment first;`)
	if err != nil{
		err=conn.Rollback()
		if err!=nil{
			this.loger.Fatal(err)
		}
		this.loger.Fatal(err)
	}
	err=conn.Commit()
	if err != nil{
		err=conn.Rollback()
		if err!=nil{
			this.loger.Fatal(err)
		}
		this.loger.Fatal()
	}
	return nil
}