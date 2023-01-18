package main

import (
	"fmt"
	"github.com/jinzaigo/xconfig"
)

func main() {
	//本地文件
	local()
	//远程apollo
	remote()
}

func local() {
	if xconfig.IsLocalLoaded() {
		fmt.Println("local config is loaded")
		return
	}
	//初始化
	configIns := xconfig.New(xconfig.WithFile("example/config.yml"))
	xconfig.InitLocalIns(configIns)

	//读取配置
	fmt.Println(xconfig.GetLocalIns().GetString("appId"))
	fmt.Println(xconfig.GetLocalIns().GetString("env"))
	fmt.Println(xconfig.GetLocalIns().GetString("apollo.one.endpoint"))

	//测试热加载
	//for {
	//	str, _ := json.Marshal(xconfig.GetLocalIns().Get("apollo"))
	//	fmt.Println(string(str))
	//	time.Sleep(10 * time.Second)
	//}
}

type ApolloConfig struct {
	Endpoint   string                     `json:"endpoint"`
	AppId      string                     `json:"appId"`
	Namespaces map[string]ApolloNameSpace `json:"namespaces"`
	BackupFile string                     `json:"backupFile"`
}

type ApolloNameSpace struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func remote() {
	if !xconfig.GetLocalIns().IsSet("apollo") {
		fmt.Println("without apollo key")
		return
	}

	//初始化
	//示例：
	//configIns := xconfig.New(xconfig.WithConfigType("properties"))
	//err := configIns.AddApolloRemoteConfig(endpoint, appId, namespace, backupFile)
	//if err != nil {
	//	...handler
	//}
	//xconfig.AddRemoteIns("ApplicationConfig", configIns)

	apolloConfigs := make(map[string]ApolloConfig, 0)
	err := xconfig.GetLocalIns().SubAndUnmarshal("apollo", &apolloConfigs)
	if err != nil {
		fmt.Println(apolloConfigs)
		fmt.Println("SubAndUnmarshal error:", err.Error())
		return
	}

	fmt.Println(apolloConfigs)

	for _, apolloConfig := range apolloConfigs {
		for _, namespaceConf := range apolloConfig.Namespaces {
			configIns := xconfig.New(xconfig.WithConfigType(namespaceConf.Type))
			err = configIns.AddApolloRemoteConfig(apolloConfig.Endpoint, apolloConfig.AppId, namespaceConf.Name, apolloConfig.BackupFile)
			if err != nil {
				fmt.Println("AddApolloRemoteConfig error:" + err.Error())
			}
			xconfig.AddRemoteIns(namespaceConf.Key, configIns)
		}
	}

	//读取
	fmt.Println(xconfig.GetRemoteIns("cipherConfig").AllSettings())

	//测试热加载
	//for {
	//	fmt.Println(xconfig.GetRemoteIns("cipherConfig").AllSettings())
	//	fmt.Println("------")
	//	time.Sleep(10 * time.Second)
	//}
}
