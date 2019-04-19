package checker

import (
	"github.com/qianlidongfeng/toolbox"
	"testing"
)

func TestChecker_Init(t *testing.T) {
	pc:=NewProxyChecker()
	appPath,err:= toolbox.AppPath()
	if err != nil{
		t.Error(err)
	}
	configFile := appPath+".yaml"
	err = pc.Init(configFile)
	if err != nil{
		t.Error(err)
	}
	pc.Start()
}