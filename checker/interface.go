package checker

type Checker interface{
	Check() error
	Init(configPath string) error
}