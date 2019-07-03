package checker

import (
	"database/sql"
	"errors"
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
	"strconv"
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
	stmtFail    *sql.Stmt
	stmtSuccess *sql.Stmt
	success     int
	selfIp      string
	selfClient  httpclient.Client
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
	fields["lastcheck"]="bigint(20) unsigned"
	fields["distributetime"]="bigint(20) unsigned"
	fields["source"]="varchar(16)" //代理来源
	fields["ctime"]="datetime" //创建时间
	err=toolbox.CheckTable(this.db,this.cfg.DB.Table,fields)
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
	this.selfClient= httpclient.NewClient()
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
	srange:=strings.Split(this.cfg.Range,"-")
	begin,_:=strconv.Atoi(srange[0])
	var end int
	if srange[1] == ""{
		end = -1
	}else{
		end,_ = strconv.Atoi(srange[1])
	}
	var left,right int
	left=begin
	for{
		this.success=0
		count:=0
		st:=time.Now()
		//rows,err:=this.stmtGet.Query()
		if end-left < this.cfg.Limit && end != -1{
			right= end - left
		}else{
			right = this.cfg.Limit
		}
		rows,err:=this.db.Query(fmt.Sprintf(`SELECT id,proxy,type,success,total,ctime FROM %s ORDER BY id LIMIT %d,%d`,this.cfg.DB.Table,left,right))
		if err != nil{
			this.loger.Fatal(err)
		}
		for rows.Next(){
			count++
			fields := Feilds{}
			err=rows.Scan(&fields.id,&fields.proxy,&fields.tp,&fields.success,&fields.total,&fields.ctime)
			if err != nil{
				this.loger.Fatal(err)
			}
			this.wg.Add(1)
			this.proxyPool<-fields
		}
		rows.Close()
		this.wg.Wait()
		err=this.DeleteInvalidProxy()
		if err != nil{
			this.loger.Fatal(err)
		}
		//this.selfIp=this.getSelfIp()
		if this.cfg.Debug{
			this.loger.Msg("totoal success",this.success)
		}
		left=left+right
		if (left>=end && end != -1)||count==0{
			left=begin
			cost := time.Since(st)/time.Second
			subt := this.cfg.Delay-cost
			if subt > 0{
				time.Sleep(subt*time.Second)
			}
		}
	}
	return nil
}

func (this *ProxyChecker)Do(){
	var fields Feilds
	fields=<-this.proxyPool
	defer this.wg.Done()
	var err error
	var resp *clientserver.Respone
	resp,err=this.GetRpcClient().Get(context.Background(),&clientserver.ProxyInfo{Proxy:fields.tp+"://"+fields.proxy})
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
	if resp.Status != 200{
		this.onFail(fields)
		return
	}
	this.onSuccess(fields)
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
	info,err := this.getProxyInfo(html)
	if err != nil{
		this.onFail(fields)
		return
	}
	if info.IP==this.selfIp{
		this.onFail(fields)
	}else{
		this.onSuccess(fields)
	}
}

func (this *ProxyChecker) Realse(){
	for _,v := range this.rpcConns{
		v.Close()
	}
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
	ctime,err:=toolbox.TimeToSecondStamp(fields.ctime)
	if err != nil{
		this.loger.Fatal(err)
	}
	now:=toolbox.GetTimeMilliStamp()
	exist:=toolbox.GetTimeSecondStamp()-ctime
	_,err=this.stmtSuccess.Exec(fields.success,math.Trunc(successrate*1e2) * 1e-2*100,fields.total,exist,1,now,fields.id)
	if err != nil{
		this.loger.Fatal(err)
	}
	this.success++
}

func (this *ProxyChecker) onFail(fields Feilds){
	fields.total++
	successrate:=float64(fields.success)/float64(fields.total)
	ctime,err:=toolbox.TimeToSecondStamp(fields.ctime)
	if err != nil{
		this.loger.Fatal(err)
	}
	now:=toolbox.GetTimeMilliStamp()
	exist:=toolbox.GetTimeSecondStamp()-ctime
	_,err=this.stmtFail.Exec(math.Trunc(successrate*1e2) * 1e-2*100,fields.total,exist,0,now,fields.id)
	if err != nil{
		this.loger.Fatal(err)
	}
}


func (this *ProxyChecker) getSelfIp() string{
	for{
		resp,err:=this.selfClient.Get("https://www.baidu.com/s?wd=ip")
		if err != nil{
			this.loger.Warn(err)
			time.Sleep(10*time.Second)
			continue
		}
		if resp.StatusCode != 200{
			this.loger.Warn(errors.New(fmt.Sprintf("get selfip failed,status code:%d",resp.StatusCode)))
			time.Sleep(10*time.Second)
			continue
		}
		info,err:=proxy.GetIpFromBaidu(resp.Html)
		if err != nil{
			this.loger.Warn(err)
			time.Sleep(time.Second*10)
			continue
		}
		return info.IP
	}
}

func (this *ProxyChecker) getProxyInfo(html string) (proxy.Info,error){
	info,err:=proxy.GetIpFromBaidu(html)
	return info,err
}

func (this *ProxyChecker) DeleteInvalidProxy() error{
	//_,err:=this.db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE successrate<? AND exist>?`,this.cfg.DB.Table),this.cfg.BadSuccessrate,this.cfg.BadExist)
	//暂时只删除搜集的，扫描的要看每个网段采集了多少
	_,err:=this.db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE successrate<? AND exist> AND source != 'scan'?`,this.cfg.DB.Table),this.cfg.BadSuccessrate,this.cfg.BadExist)
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