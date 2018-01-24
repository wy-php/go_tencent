package model

import (
  "tda/config"

  "fmt"
  "time"
 // "strings"
 // "encoding/json"

  "github.com/jinzhu/gorm"
  "github.com/go-redis/redis"
  "github.com/eclipse/paho.mqtt.golang"
  log "github.com/sirupsen/logrus"
)

const (
  // 设备应答
  MqttAckTopic         = "/v1/polyhome-ha/host/+/dev_into_zigbee/"
  MqttSubTimingTopic  = "/v1/polyhome-ha/host/+/state_change/"
)

// 声明相关client全局变量
var RedisClient *redis.Client;
var MqttClient mqtt.Client;
var DB *gorm.DB;

func initDB() {
  // connect mysql
  dbURI := fmt.Sprintf("%s:%s@/%s?charset=%s&parseTime=True",
    config.DB.Username,
    config.DB.Password,
    config.DB.Name,
    config.DB.Charset,
  )

  db, err := gorm.Open(config.DB.Dialect, dbURI)
  if err != nil {
    log.Fatal("Could not connect database")
  }

  DB = db
}

func initRedis() {
  log.Info("connect redis...")
  RedisClient = redis.NewClient(&redis.Options{
    Addr:           config.Redis.Addr,
    Password:       config.Redis.Password,
    DB:             config.Redis.DB,
    PoolSize:       config.Redis.PoolSize,
  })

  _, redisErr := RedisClient.Ping().Result()

  if redisErr == nil {
    log.Info("connect redis success!")
  }else{
    log.Fatal("connect redis fail!")
  }
}

func initMqtt(f mqtt.MessageHandler) {
  log.Info("connect mqtt ...")
  clientId := "GoApi_" + fmt.Sprintf("%d", time.Now().Unix())
  // mqtt server
  ops := mqtt.NewClientOptions()
  ops.AddBroker(config.Mqtt.Broker)
  ops.SetClientID(clientId)
  ops.SetUsername(config.Mqtt.Username)
  ops.SetPassword(config.Mqtt.Password)

  ops.SetDefaultPublishHandler(f)

  MqttClient = mqtt.NewClient(ops)

  // check connect is normal.
  if token := MqttClient.Connect(); token.Wait() && token.Error() != nil {
    log.Error(token.Error())
  }

  // subscribe topic
  // params: 1. topic 2. qos 3. callback
  // source: https://github.com/eclipse/paho.mqtt.golang/blob/master/client.go
  // MqttClient.Subscribe(MqttSubStaChgTopic, 0, nil)
  MqttClient.Subscribe(MqttSubTimingTopic, 0, nil)
  MqttClient.Subscribe(MqttAckTopic, 2, nil)

  log.Info("mqtt connect success.")
}

func Init(f mqtt.MessageHandler) {
  initDB()
  initRedis()
  initMqtt(f)
}
