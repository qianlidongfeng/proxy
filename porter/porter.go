package porter

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

type ProxyPorter struct{
	srcdb          *sql.DB
	destdb          *sql.DB
	cfg         Config
	loger       loger.Loger
	stdout      *os.File
	rpcConns     []*grpc.ClientConn
	rpcClients   []clientserver.HttpClient
	proxyPool   chan Feilds
	selfIp      string
	selfClient  httpclient.Client
	clientIndex int
	mu sync.Mutex
	wg sync.WaitGroup
}

func NewProxyPorter() ProxyPorter{
	return ProxyPorter{}
}

func (this *ProxyPorter) Init(configPath string) error{
	err:=toolbox.LoadConfig(configPath,&this.cfg)
	if err != nil{
		return err
	}
	if this.cfg.Debug == false{
		this.stdout, err = toolbox.RediRectOutPutToLog()
		if err != nil{
			log.Fatal(err)
		}
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
	this.srcdb,err= toolbox.InitMysql(this.cfg.SrcDB)
	if err != nil{
		this.loger.Fatal(err)
	}
	this.destdb,err= toolbox.InitMysql(this.cfg.DestDB)
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

func (this *ProxyPorter) GetRpcClient() clientserver.HttpClient{
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.clientIndex >= len(this.rpcClients){
		this.clientIndex=0
	}
	client:=this.rpcClients[this.clientIndex]
	this.clientIndex++
	return client
}

func (this *ProxyPorter) Port() error{
	//this.selfIp=this.getSelfIp()
	fmt.Println("begin")
	for i:=0;i<this.cfg.Thread;i++{
		go func(index int){
			for{
				this.Do()
			}
		}(i)
	}
	for{
		rows,err:=this.srcdb.Query(fmt.Sprintf(`select id,proxy,type,source,ctime,iprange from %s order by id limit ?`,this.cfg.SrcDB.Table),this.cfg.Limit)
		if err != nil{
			this.loger.Fatal(err)
		}
		var ids []int64
		for rows.Next(){
			fields := Feilds{}
			var id int64
			err=rows.Scan(&id,&fields.proxy,&fields.tp,&fields.source,&fields.ctime,&fields.iprange)
			if err != nil{
				this.loger.Fatal(err)
			}
			ids= append(ids,id)
			this.proxyPool<-fields
			this.wg.Add(1)
		}
		rows.Close()
		this.wg.Wait()
		idCount :=len(ids)
		if idCount != 0{
			var sids string
			for i:=0;;i++{
				if i>=idCount{
					if  sids !=""{
						sids=strings.TrimRight(sids,",")
						_,err=this.srcdb.Exec(fmt.Sprintf(`delete from %s where id in (%s)`,this.cfg.SrcDB.Table,sids))
						if err != nil{
							this.loger.Msg("sql",fmt.Sprintf(`delete from %s where id in (%s)`,this.cfg.SrcDB.Table,sids))
							this.loger.Fatal(err)
						}
						sids=""
					}
					break
				}
				sids=sids+strconv.FormatInt(ids[i],10)+","
				if (i+1)%500 == 0 && sids !=""{
					sids=strings.TrimRight(sids,",")
					_,err=this.srcdb.Exec(fmt.Sprintf(`delete from %s where id in (%s)`,this.cfg.SrcDB.Table,sids))
					if err != nil{
						this.loger.Msg("sql",fmt.Sprintf(`delete from %s where id in (%s)`,this.cfg.SrcDB.Table,sids))
						this.loger.Fatal(err)
					}
					sids=""
				}
			}
		}else{
			time.Sleep(time.Second)
		}
		//this.selfIp=this.getSelfIp()
		time.Sleep(time.Millisecond*this.cfg.Delay)
	}
	return nil
}

type Feilds struct{
	proxy string
	port int
	area string
	tp string
	category int
	source string
	ctime string
	iprange string
}

func (this *ProxyPorter) Do(){
	var fields Feilds
	fields=<-this.proxyPool
	defer this.wg.Done()
	var err error
	var resp *clientserver.Respone
	client:= this.GetRpcClient()
	if strings.ToLower(fields.tp)=="http" || strings.ToLower(fields.tp)=="https"{
		//Url:"https://www.baidu.com/s?wd=ip"
		resp,err=client.Get(context.Background(),&clientserver.ProxyInfo{Proxy:"http://"+fields.proxy,})
		fields.category=0
	}else if strings.ToLower(fields.tp)=="socks5"{
		resp,err=client.Get(context.Background(),&clientserver.ProxyInfo{Proxy:"socks5://"+fields.proxy})
		fields.category=0
	}else if strings.ToLower(fields.tp)=="unknown"{
		resp,err=client.Get(context.Background(),&clientserver.ProxyInfo{Proxy:"http://"+fields.proxy})
		if err != nil || resp == nil{
			resp,err=client.Get(context.Background(),&clientserver.ProxyInfo{Proxy:"socks5://"+fields.proxy})
			if err == nil && resp != nil{
				fields.tp="socks5"
				fields.category=0
			}
		}else{
			fields.tp="http"
			fields.category=0
		}
	}
	if err != nil || resp == nil{
		code:=status.Code(err)
		if code != codes.Unknown && code !=codes.OK{
			this.loger.Warn(fmt.Sprintf("error code:%d, message:%s",code,err.Error()))
			time.Sleep(time.Second*60)
			return
		}
		return
	}
	if resp.Status != 200{
		return
	}
	this.onSuccess(fields)
}

func (this *ProxyPorter) Parse(html string,fields Feilds){
	info,err := this.getProxyInfo(html)
	if err != nil{
		return
	}
	if info.IP==this.selfIp{
		return
	}else{
		fields.area=info.Area
		this.onSuccess(fields)
	}
}

func (this *ProxyPorter) Realse(){
	for _,v := range this.rpcConns{
		v.Close()
	}
	this.destdb.Close()
	this.srcdb.Close()
	this.stdout.Close()
	this.loger.Close()
}

func (this *ProxyPorter) checkRpcReady(conn *grpc.ClientConn,timeout time.Duration) bool{
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var state connectivity.State
	for state = conn.GetState(); state != connectivity.Ready && conn.WaitForStateChange(ctx, state); state = conn.GetState() {
	}
	return state == connectivity.Ready
}

func (this *ProxyPorter) onSuccess(fields Feilds){
	fields.port,_= strconv.Atoi(strings.Split(fields.proxy,":")[1])
	_,err:=this.destdb.Exec(fmt.Sprintf(`insert ignore into %s (proxy,port,type,category,source,ctime,iprange)values(?,?,?,?,?,?,?)`,this.cfg.DestDB.Table),fields.proxy,fields.port,fields.tp,fields.category,fields.source,fields.ctime,fields.iprange)
	//&& err.(*mysql.MySQLError).Number != 1062
	if err != nil{
		this.loger.Warn(err)
	}
}


func (this *ProxyPorter) getSelfIp() string{
	for{
		resp,err:=this.selfClient.Get("http://ip.cz88.net/data.php")
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
		info,err:=proxy.GetIpFromCz(resp.Html)
		if err != nil{
			this.loger.Warn(err)
			time.Sleep(time.Second*10)
			continue
		}
		return info.IP
	}
}


func (this *ProxyPorter) getProxyInfo(html string) (proxy.Info,error){
	info,err:=proxy.GetIpFromBaidu(html)
	return info,err
}

