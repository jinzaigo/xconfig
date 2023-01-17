package xconfig

import (
	"fmt"
	"log"
	"sync"
)

var (
	localIns  *Config  //本地配置实例
	remoteIns sync.Map //远程配置实例（支持多个，key为唯一字符串，value为*Config）
)

func IsLocalLoaded() bool {
	return localIns != nil
}

func InitLocalIns(c *Config) {
	localIns = c
}

func GetLocalIns() *Config {
	//未初始化则退出
	if !IsLocalLoaded() {
		log.Panic("本地配置实例未初始化")
	}
	return localIns
}

func AddRemoteIns(key string, ins *Config) {
	remoteIns.Store(key, ins)
}

func GetRemoteIns(key string) *Config {
	v, ok := remoteIns.Load(key)
	if ok {
		return v.(*Config)
	}
	log.Panic(fmt.Sprintf("远程配置实例%s未初始化", key))
	return nil
}
