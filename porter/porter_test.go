package porter

import (
	"github.com/qianlidongfeng/toolbox"
	"testing"
)

func TestNewProxyChecker(t *testing.T) {
	pt:=NewProxyPorter()
	configFile,err:= toolbox.GetConfigFile()
	if err != nil{
		panic(err)
	}
	err = pt.Init(configFile)
	pt.Port()
}