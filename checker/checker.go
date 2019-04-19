package checker

import (
	"database/sql"
	"fmt"
	"github.com/qianlidongfeng/httpclient"
	"github.com/qianlidongfeng/loger"
	"github.com/qianlidongfeng/loger/netloger"
	"github.com/qianlidongfeng/toolbox"
	"log"
	"math"
	"os"
	"sync"
	"time"
)

type ProxyChecker struct{
	db *sql.DB
	cfg Config
	loger loger.Loger
	stdout *os.File
	clients []httpclient.HttpClient
	proxyPool chan Feilds
	stmtGet *sql.Stmt
	stmtFail *sql.Stmt
	stmtSuccess *sql.Stmt
	finish bool
	url string
	index int
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
			this.loger.Fatal(err)
		}
		this.stdout, err = os.OpenFile(appPath+".log", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0644)
		if err != nil{
			this.loger.Fatal(err)
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
	fields["city"]="varchar(64)"
	fields["type"]="varchar(16)" //代理类型
	fields["category"]="varchar(16)" //匿名度
	fields["status"]="tinyint(4) unsigned" //代理状态
	fields["success"]="int(11) unsigned"  //成功总次数
	fields["total"]="int(11) unsigned"//检测总次数
	fields["successrate"]="tinyint(4) unsigned"//成功率
	fields["exist"]="int(11) unsigned" //存在总时长
	fields["ctime"]="datetime" //创建时间
	err=toolbox.CheckTable(this.db,this.cfg.DB.Table,fields)
	if err != nil{
		this.loger.Fatal(err)
	}
	this.stmtGet,err=this.db.Prepare(fmt.Sprintf(`SELECT id,proxy,success,total,ctime FROM %s WHERE id>? limit ?`,this.cfg.DB.Table))
	if err != nil{
		this.loger.Fatal(err)
	}
	this.stmtSuccess,err=this.db.Prepare(fmt.Sprintf(`UPDATE %s SET success=?,successrate=?,total=?,exist=?,status=? WHERE id=?`,this.cfg.DB.Table))
	if err != nil{
		this.loger.Fatal(err)
	}
	this.stmtFail,err=this.db.Prepare(fmt.Sprintf(`UPDATE %s SET successrate=?,total=?,exist=?,status=? WHERE id=?`,this.cfg.DB.Table))
	if err != nil{
		this.loger.Fatal(err)
	}
	for i:=0;i<this.cfg.Thread;i++{
		client,err:= httpclient.NewHttpClient()
		if err != nil{
			log.Fatal(err)
		}
		client.SetTimeOut(this.cfg.TimeOut*time.Millisecond)
		this.clients= append(this.clients,client)
	}
	this.proxyPool = make(chan Feilds,this.cfg.Thread)
	this.url="http://2019.ip138.com/ic.asp"
	return nil
}

func (this *ProxyChecker) Check() error{

	fmt.Println()
	return nil
}

func (this *ProxyChecker) Start() error{
	wg := sync.WaitGroup{}
	for i:=0;i<this.cfg.Thread;i++{
		wg.Add(1)
		go func(index int){
			for{
				var fields Feilds
				select{
				case fields=<-this.proxyPool:
					if this.finish{
						this.finish=false
						wg.Add(1)
					}
				case <-time.After(time.Second*3):
					if this.finish == false{
						this.finish=true
						wg.Done()
					}
					continue
				}
				/*_=fields
				time.Sleep(time.Millisecond*100)
				_,err:=this.db.Exec(fmt.Sprintf(`UPDATE %s SET successrate=%d,total=%d,exist=%d,status=%d WHERE id=%d`,this.cfg.DB.Table,10,1,0,0,1))
				if err != nil{
					this.db.Close()
					this.loger.Fatal(err)
				}*/
				this.clients[index].SetHttpProxy("http://"+fields.proxy)
				html,err:=this.clients[index].Get(this.url)
				if err != nil{
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
					_,err=this.stmtFail.Exec(math.Trunc(successrate*1e2) * 1e-2*100,fields.total,now.Unix()-ctime.Unix(),0,fields.id)
					//_,err=this.db.Exec(fmt.Sprintf(`UPDATE %s SET successrate=%d,total=%d,exist=%d,status=%d WHERE id=%d`,this.cfg.DB.Table,10,1,0,0,1))
					if err != nil{
						this.loger.Fatal(err)
					}
					continue
				}
				this.index++
				this.Parse(html,fields)
			}
		}(i)
	}
	index:=0
	for{
		rows,err:=this.stmtGet.Query(index,this.cfg.Limit)
		if err != nil{
			this.loger.Fatal(err)
		}
		cnt:=0
		for rows.Next(){
			fields := Feilds{}
			err=rows.Scan(&fields.id,&fields.proxy,&fields.success,&fields.total,&fields.ctime)
			if err != nil{
				this.loger.Fatal(err)
			}
			this.proxyPool<-fields
			index++
			cnt++
		}
		if cnt == 0{
			fmt.Println(this.index)
			this.index=0
			index=0
		}
		rows.Close()
		time.Sleep(this.cfg.TimeOut)
	}
	wg.Wait()

	return nil
}

type Feilds struct{
	id int64
	proxy string
	success int
	total int
	ctime string
}

func (this *ProxyChecker) Parse(html string,fields Feilds){
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
	_,err=this.db.Exec(fmt.Sprintf(`INSERT INTO temp SET proxy="%s"`,fields.proxy))
	if err != nil{
		fmt.Println(err)
	}
	//_,_,_=successrate,ctime,now
	_,err=this.stmtSuccess.Exec(fields.success,math.Trunc(successrate*1e2) * 1e-2*100,fields.total,now.Unix()-ctime.Unix(),1,fields.id)
	//_,err=this.stmtSuccess.Exec(1,10,1,1,1,1)
	if err != nil{
		this.loger.Fatal(err)
	}
}

func (this *ProxyChecker) Realse(){
	this.stmtGet.Close()
	this.stmtFail.Close()
	this.stmtSuccess.Close()
	this.db.Close()
	this.stdout.Close()
	this.loger.Close()
}

func (this *ProxyChecker) BadProxy(feilds Feilds){

}
