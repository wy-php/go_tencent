package app

import (
  "tda/app/handler"
  "tda/app/model"
  "tda/config"

  "net/http"
  "os"
  "encoding/json"
  "strings"
  "fmt"
  "strconv"
  _ "bytes"

  "github.com/go-redis/redis"
  "github.com/eclipse/paho.mqtt.golang"
  "github.com/jinzhu/gorm"
  "github.com/gorilla/mux"
  log "github.com/sirupsen/logrus"
)

type App struct {
  Router    *mux.Router
  DB        *gorm.DB
  Redis     *redis.Client
  Mqtt      mqtt.Client
}

// Initialize the app with predefined configuration
func (app *App) Initialize() {
   //set log info
  log.SetOutput(os.Stdout)
  log.SetLevel(log.InfoLevel)
  log.SetFormatter(&log.TextFormatter{})

  config.GetConfig()

   // msg source code: https://godoc.org/github.com/eclipse/paho.mqtt.golang#Message
  var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    log.WithFields(log.Fields{
      "MSG": string(msg.Payload()),
      "TOPIC": string(msg.Topic()),
    }).Info("[MQTT] Receive message: ")
    // 注意: 带下划线的key json.Unmarshal不能和struct中的key对应
    //fmtMsg := bytes.Replace(msg.Payload(), []byte("entity_id"), []byte("entityId"), -1)
    newMsg := new(model.Payload)
    err := json.Unmarshal(msg.Payload(), newMsg)
    if err != nil {
      log.Error("[MQTT] json unmarshal error", err)
    }
    topicType := strings.Split(string(msg.Topic()), "/")[5]
    switch topicType {
    case "state_change", "heart_beat":
      entityId := newMsg.Data["entity_id"]
      if entityId != nil {
        sn := strings.Split(msg.Topic(), "/")[4]
        handler.SaveDeviceInfo(entityId.(string), sn, newMsg)
      } else {
        log.Error("[MQTT] The data['entity_id'] is nil!")
      }
    case "dev_into_zigbee":
      entityId := newMsg.Data["entity_id"].(string)
      sn := strings.Split(msg.Topic(), "/")[4]
      parentDin, token := "", ""
      // 设备注册, sn和entityId一致则是网关注册，网关的话就可以直接去获取token，如果是网关下的设备可以直接从redis中取设备，如果redis中过期则重新注册生成
      if sn == entityId {
        token = handler.TxLogin()["token"].(string)
      } else {
        var arrs []string
        model.DB.Find(&model.Device{Sn: sn}).Pluck("parent_din", &arrs)
        parentDin = arrs[0]
        token = handler.GetToken(parentDin)
      }
      // 多个设备是同一种类型的时候使用
      deviceType := newMsg.Data["device_type"]
      if deviceType == nil {
        log.Error("[MQTT] The data['device_type' is nil]")
        return
      }
      // 将unicode编码转成utf-8
      name := fmt.Sprint(newMsg.Data["friendly_name"].(string))
      dType := strconv.Itoa(handler.FindDtypeById(entityId, deviceType.(string)))
      handler.TxDeviceRegister(token, dType, parentDin, entityId, name)
    default:
      log.Error("[MQTT] topic[%s] error", topicType)
    }
  }

  model.Init(f)

  app.DB = model.DBMigrate(model.DB)

  // set router
  app.Router = mux.NewRouter()
  app.setRouters()

  log.Info("server start success.")
}

func (app *App) setRouters() {
  app.Post("/iotd/ctl/spController", app.SpController)
  app.Get("/iotd/device/spGetDeviceStatus", app.SpGetDeviceStatus)
}


// Get wraps the router for GET method
func (app *App) Get(path string, f func(w http.ResponseWriter, r *http.Request)) {
  app.Router.HandleFunc(path, f).Methods("GET")
}

// Post wraps the router for POST method
func (app *App) Post(path string, f func(w http.ResponseWriter, r *http.Request)) {
  app.Router.HandleFunc(path, f).Methods("POST")
}

func (app *App) SpController(w http.ResponseWriter, r *http.Request) {
  handler.SpController(w, r)
}

func (app *App) SpGetDeviceStatus(w http.ResponseWriter, r *http.Request) {
  handler.SpGetDeviceStatus(w, r)
}

func (app *App) Run(host string) {
  log.Fatal(http.ListenAndServe(host, app.Router))
}
