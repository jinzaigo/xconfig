# xconfig
golang基于[spf13/viper](https://github.com/spf13/viper) 和 [shima-park/agollo](https://github.com/shima-park/agollo) 实现本地配置文件和远程apollo配置中心多实例快速接入

## 快速开始
### 获取安装
```
go get -u github.com/jinzaigo/xconfig
```

## Features
* 支持viper包诸多同名方法
* 支持本地配置文件和远程apollo配置热加载，实时更新
* 使用sync.RWMutex读写锁，解决了viper并发读写不安全问题
* 支持apollo配置中心多实例配置化快速接入

## 接入示例

### 本地配置文件
指定配置文件路径完成初始化，即可通过xconfig.GetLocalIns().xxx()链式操作，读取配置
```go
package main

import (
    "fmt"
    "github.com/jinzaigo/xconfig"
)

func main() {
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
}
```

xxx支持的操作方法：

- IsSet(key string) bool 
- Get(key string) interface{} 
- AllSettings() map[string]interface{} 
- GetStringMap(key string) map[string]interface{} 
- GetStringMapString(key string) map[string]string 
- GetStringSlice(key string) []string 
- GetIntSlice(key string) []int 
- GetString(key string) string 
- GetInt(key string) int 
- GetInt32(key string) int32 
- GetInt64(key string) int64 
- GetUint(key string) uint 
- GetUint32(key string) uint32 
- GetUint64(key string) uint64 
- GetFloat(key string) float64 
- GetFloat64(key string) float64 
- GetFloat32(key string) float32 
- GetBool(key string) bool 
- SubAndUnmarshal(key string, i interface{}) error 

### 远程apollo配置中心

指定配置类型与apollo信息完成初始化，即可通过xconfig.GetRemoteIns(key).xxx()链式操作，读取配置

单实例场景
```go
//初始化
configIns := xconfig.New(xconfig.WithConfigType("properties"))
err := configIns.AddApolloRemoteConfig(endpoint, appId, namespace, backupFile)
if err != nil {
    ...handler
}
xconfig.AddRemoteIns("ApplicationConfig", configIns)

//读取配置
fmt.Println(xconfig.GetRemoteIns("ApplicationConfig").AllSettings())
```

多实例场景

在本地配置文件config.yaml维护apollo配置信息，然后批量完成多个实例的初始化，即可通过xconfig.GetRemoteIns(key).xxx()链式操作，读取配置

```yaml
#apollo配置，支持多实例多namespace
apollo:
  one:
    endpoint: xxx
    appId: xxx
    namespaces:
      one:
        key: ApplicationConfig   #用于读取配置，保证全局唯一，避免相互覆盖
        name: application        #注意：name不要带类型（例如application.properties），这里name和type分开配置
        type: properties
      two:
        key: cipherConfig
        name: cipher
        type: properties
    backupFile: /tmp/xconfig/apollo_bak/test.agollo #每个appId使用不同的备份文件名，避免相互覆盖
```

```go
package main

import (
    "fmt"
    "github.com/jinzaigo/xconfig"
)

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

func main() {
    //本地配置初始化
    xconfig.InitLocalIns(xconfig.New(xconfig.WithFile("example/config.yml")))
    if !xconfig.GetLocalIns().IsSet("apollo") {
        fmt.Println("without apollo key")
        return
    }

    apolloConfigs := make(map[string]ApolloConfig, 0)
    err := xconfig.GetLocalIns().SubAndUnmarshal("apollo", &apolloConfigs)
    if err != nil {
        fmt.Println(apolloConfigs)
        fmt.Println("SubAndUnmarshal error:", err.Error())
        return
    }

    //多实例初始化
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
    fmt.Println(xconfig.GetRemoteIns("ApplicationConfig").AllSettings())
}

```

