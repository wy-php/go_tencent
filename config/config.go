package config

import (
  "time"
  "fmt"
)

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
  DB = &DBConfig{
    Dialect:  "mysql",
    Username: "root",
//    Password: "Uo0BhmWldZBTBjgP",
    Password: "",
    Name:     "tda",
    Charset:  "utf8",
  }
  Redis = &RedisConfig{
    Addr:         "localhost:6379",
    Password:     "",
    DB:           1,
    DialTimeout:  10 * time.Second,
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 30 * time.Second,
    PoolSize:     10,
    PoolTimeout:  30 * time.Second,
  }
  Mqtt = &MqttConfig{
    Broker:   "tcp://123.57.139.200:1883",
    ClientId: "GoApi_" + fmt.Sprintf("%d", time.Now().Unix()),
    Username: "polyhome",
    Password: "123",
  }
}

//func GetConfig() *Config {
//  return &Config{
//    DB: &DBConfig{
//      Dialect:  "mysql",
//      Username: "root",
//      Password: "",
//      Name:     "tda",
//      Charset:  "utf8",
//    },
//    Redis: &RedisConfig{
//      Addr:         "localhost:6379",
//      Password:     "",
//      DB:           1,
//      DialTimeout:  10 * time.Second,
//      ReadTimeout:  30 * time.Second,
//      WriteTimeout: 30 * time.Second,
//      PoolSize:     10,
//      PoolTimeout:  30 * time.Second,
//    },
//    Mqtt: &MqttConfig{
//      Broker:   "tcp://123.57.139.200:1883",
//      ClientId: "GoApi_" + fmt.Sprintf("%d", time.Now().Unix()),
//      Username: "polyhome",
//      Password: "123",
//    },
//  }
//}
