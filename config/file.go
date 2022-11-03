package config

import (
	"github.com/spf13/viper"
	"log"
	"os"
)

type ProxyConf struct {
	Name     string
	PType    string `mapstructure:"type"`
	Server   string
	Port     int
	Cipher   string
	Password string
}

type Config struct {
	Host    string
	Port    int
	Proxies []ProxyConf
}

var config Config

var (
	ConfigPath string
)

func init() {
	ConfigPath, _ = os.Getwd()
	viper.SetDefault("host", "127.0.0.1")
	viper.SetDefault("port", 7890)
}

func ReadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(ConfigPath)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	err := viper.Unmarshal(&config)
	if err != nil {
		log.Println(err)
		return nil
	}
	return nil
}

func GetProxyConf() *ProxyConf {
	if len(config.Proxies) == 0 {
		return nil
	}
	return &config.Proxies[0]
}
