package config

import (
  "time"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "os"
  "regexp"

  "tda/app/utils"
)

var jsonData map[string]interface{}

func initJSON() {
  bytes, err := ioutil.ReadFile("./config.json")
  if err != nil {
    fmt.Println("ReadFile: ", err.Error())
    os.Exit(-1)
  }

  configStr := string(bytes[:])
  reg := regexp.MustCompile(`/\*.*\*/`)

  configStr = reg.ReplaceAllString(configStr, "")
  bytes = []byte(configStr)

  if err := json.Unmarshal(bytes, &jsonData); err != nil {
    fmt.Println("invalid config: ", err.Error())
    os.Exit(-1)
  }
}

const (
  RedisHost         = "localhost"
  RedisPort         = "6379"
)

var DB    *DBConfig;
var Redis *RedisConfig;
var Mqtt  *MqttConfig;

type DBConfig struct {
  Dialect  string
  Username string
  Password string
  Name     string
  Charset  string
}

type RedisConfig struct {
  Addr            string
  Password        string
  DB              int
  PoolSize        int
  PoolTimeout     time.Duration
  DialTimeout     time.Duration
  ReadTimeout     time.Duration
  WriteTimeout    time.Duration
}

type MqttConfig struct {
  Broker          string
  ClientId        string
  Username        string
  Password        string
}

func GetConfig() {
  initJSON()
  var dbConfig DBConfig
  var redis RedisConfig
  var mqtt MqttConfig
  utils.SetStructByJSON(&dbConfig, jsonData["database"].(map[string]interface{}))
  utils.SetStructByJSON(&redis, jsonData["redis"].(map[string]interface{}))
  utils.SetStructByJSON(&mqtt, jsonData["mqtt"].(map[string]interface{}))
  DB = &dbConfig
  Redis = &redis
  Mqtt = &mqtt
}

