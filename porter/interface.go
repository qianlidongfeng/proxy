package porter

type Porter interface{
	Port() error
	Init(configPath string) error
}