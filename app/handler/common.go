package handler

import (
  "tda/app/model"
  _ "tda/config"

  _ "net/http"
  "fmt"
  "bytes"
  "encoding/binary"
  "strings"
  "strconv"
  "time"
  "crypto/sha1"
  _ "reflect"
  "encoding/json"

  _ "github.com/go-redis/redis"
  _ "github.com/eclipse/paho.mqtt.golang"
  _ "github.com/jinzhu/gorm"
  _ "github.com/gorilla/mux"
  log "github.com/sirupsen/logrus"
)

var db = model.DB;

const (
   AppId             = 22
   AppKey            = "gGQPKB1h"
   SpAppId           = "6023"
   SpSkey            = "polyhome"
   Name              = "北京博力恒昌科技有限公司"
   Desc              = "一家力图改变您的生活方式的智能家居公司。"
   Url               = "https://qq.ourjujia.com/iotd/ctl/spController"
   RandNum           = 3424901
   RedisKeyPrefix    = "PHA::"
   GateWayTokenKey   = "GATEWAY::TOKEN::"
   DeviceKey         = "DEVICE::"
   Second            = time.Second
   Minute            = 60 * Second
   Hour              = 60 * Minute
   Day               = 24 * Hour
   Week              = 7  * Day
   Gateway           = 20002
   Light             = 20003
   Switch            = 20004
   PolyPanel4        = 20005
   BinarySensor      = 20006
   Cover             = 20007
   PolyPirSensor     = 20008
   LightPointId      = "1000202"
   SwitchPointId     = "1000204"
   CoverPointId      = "1000207"
   BinarySensorPointId     = "1000206"
   PolyPanel4PointId = "1000205"
   GatewayText       = "gateway"
   LightText         = "light"
   SwitchText        = "switch"
   CoverText         = "cover"
   BinarySensorText  = "binary_sensor"
   PolyPirSensorText = "polypirsensor"
   PolyPanel4Text    = "polypanel4"
   Offline           = "unavailable"
)

var Plugins = map[uint64]interface{}{
  Light: LightText,
  Switch: SwitchText,
  Cover: CoverText,
  BinarySensor: BinarySensorText,
}

var Plugins1 = map[string]interface{}{
  LightText: Light,
  SwitchText: Switch,
  CoverText: Cover,
  GatewayText: Gateway,
  BinarySensorText: BinarySensor,
  PolyPirSensorText: PolyPirSensor,
  PolyPanel4Text: PolyPanel4,
}

var loc, _ = time.LoadLocation("Asia/Shanghai")

func FindDtypeById(entityId string, deviceType string) int {
  prefix := strings.Split(entityId, ".")[0]
  dType := Plugins1[prefix]

  if deviceType == PolyPirSensorText || deviceType == PolyPanel4Text || deviceType == GatewayText {
    dType = Plugins1[deviceType]
  } else if dType == nil {
    log.Error("Type error")
    return 0
  }

  return dType.(int)
}

// generate token key
// the key is parentDin or sn
// used prefix differentiate gateway or device
func genTokenKey(key string, prefix string) string {
  return RedisKeyPrefix + prefix + key
}

// 生成token
func genToken(key string) string {
  // key
  keyByte := []byte(key)
  // time
  t := time.Now().Unix()
  timeBuf := bytes.NewBuffer([]byte{})
  binary.Write(timeBuf, binary.LittleEndian, t)
  // join
  all := [][]byte{keyByte, timeBuf.Bytes()}
  data := bytes.Join(all, []byte(""))
  hash := sha1.New()
  hash.Write(data)
  return fmt.Sprintf("%X", hash.Sum(nil))
}

// save token to redis server
func saveToken(token string, timeNum int64, parentDin string) {
  timeData := time.Unix(timeNum, 0).In(loc)
  now := time.Now().In(loc)
  expireTime := timeData.Sub(now)
  model.RedisClient.Set(genTokenKey(parentDin, GateWayTokenKey), token, expireTime)
  log.WithFields(log.Fields{
    "patentDin": parentDin,
    "token": token,
  }).Info("Save token to redis server.")
}

// get token from redis server
func GetToken(parentDin string) string {
  key := genTokenKey(parentDin, GateWayTokenKey)
  token, err := model.RedisClient.Get(key).Result()
  log.WithFields(log.Fields{
    "key": key,
    "parendDin": parentDin,
    "token": token,
  }).Info("get token by parentDin")
  if token == "" || parentDin == "" {
    // get a new token
    data := TxLogin()
    if data["token"] == nil {
      log.Error("[TXAPI] Get token error")
    }
    token = data["token"].(string)
    saveToken(data["token"].(string), data["expiryTime"].(int64), parentDin)
  }else if err != nil {
    log.Error("Get redis value error", err)
  }else{
  }
  return token
}


// 加密
func encrypt(skey string, time int64, num int64) string {
  //key
  keyByte := []byte(skey)
  //time
  timeBuf := bytes.NewBuffer([]byte{})
  binary.Write(timeBuf, binary.LittleEndian, time)
  //rand
  randBuf := bytes.NewBuffer([]byte{})
  binary.Write(randBuf, binary.LittleEndian, num)
  //join
  all := [][]byte{keyByte, timeBuf.Bytes(), randBuf.Bytes()}
  data := bytes.Join(all, []byte(""))
  //gen sign
  hash := sha1.New()
  hash.Write(data)
  return fmt.Sprintf("%X", hash.Sum(nil))
}

func formatData(data []byte, key string) interface{} {
  var objmap map[string]interface{}
  err := json.Unmarshal(data, &objmap)
  if err != nil {
    log.Error(err)
  }
  data1 := objmap["data"]
  if data1 == nil {
    return ""
  }
  data2 := data1.(map[string]interface{})
  return data2[key]
}

// 格式化返回值
func formatResult(result *model.Result) interface{} {

  params := map[string]interface{}{}
  // all device type

  //dtypeNum, _ := strconv.ParseInt(result.Dtype, 10, 64)
  params["plugin"] = Plugins[result.Dtype]
  params["data"] =  map[string]interface{}{}
  params["service"] = "turn_off"
  result.Cmd["entity_id"] = result.Sn
  delete(result.Cmd, "datapointId")
  delete(result.Cmd, "button")
  switch result.Dtype{
  case 20003:
    if result.Cmd["on"] == true || result.Cmd["on"] == "true" {
      params["service"] = "turn_on"
      snStr := strings.SplitAfter(result.Sn, ".")[1]
      if strings.SplitAfter(snStr, "dimlight")[0] == "dimlight" {
        result.Cmd["brightness"] = result.Cmd["bright"]
      }
    }
    delete(result.Cmd, "on")
    delete(result.Cmd, "bright")
    params["data"] = result.Cmd
  case 20004:
    if result.Cmd["switch"] == true {
      params["service"] = "turn_on"
    }
    delete(result.Cmd, "switch")
    params["data"] = result.Cmd
  case 20007:
    switch result.Cmd["action"]{
      case "open":
        params["service"] = "open_cover"
      case "stop":
        params["service"] = "stop_cover"
      case "close":
        params["service"] = "close_cover"
      default:
        params = map[string]interface{}{}
        log.Error("please input correct params[action] in cmd")
    }
    delete(result.Cmd, "action")
    params["data"] = result.Cmd
  default:
    params = map[string]interface{}{}
    log.Error("please input correct params[dtype]")
  }
  return params
}

// 保存控制信息 在上报消息的时候使用
func saveControlInfo(result *model.Result){
  log.WithFields(log.Fields{
    "result": result,
  }).Info("Call save control info")

  if result.Dtype ==  Light {

    // 设备控制相关信息
    cmd := map[string]interface{}{
      "button": result.Cmd["button"],
      "on": result.Cmd["on"],
      "bright": result.Cmd["bright"],
    }

    // status表示所有的按键状态
    status := [1]map[string]interface{}{}

    status[0] = cmd

    msg := map[string]interface{}{
      "datapointId": result.Cmd["datapointId"],
      "click": 1, // 1表示触发上报 0表示定时上报
      "status":  status,
    }

    result.Cmd = msg

  }

  switch result.Dtype {
  case Light:
    result.Cmd["datapointId"] = LightPointId
  case Switch:
    result.Cmd["datapointId"] = SwitchPointId
  case Cover:
    result.Cmd["datapointId"] = CoverPointId
  default:
    log.Error("Device type error", result.Dtype)
  }

  cmd, _ := json.Marshal(result.Cmd)

  // refresh new token·
  token := GetToken(result.ParentDin)

  controlInfo := map[string]interface{}{
    "token": token,
    "din": result.Din,
    "dType": result.Dtype,
    "parentDin": result.ParentDin, // 用于获取网关token
    "timestamp": result.Timestamp,
    "msg": cmd,
  }

  // return a string info, but regardless of success or fail.
  resultInfo := model.RedisClient.HMSet(genTokenKey(result.ParentDin + result.Sn, DeviceKey), controlInfo)

  if resultInfo != nil {
    log.Info("[REDIS] set ctontrol info to redis server: ", resultInfo)
  }

}

func SaveDeviceInfo(entityId string, gateWaySn string, data *model.Payload) {

  // 查询parentDin
  dev := model.Device{}

  _ = model.DB.First(&dev, model.Device{Sn: gateWaySn})

  device := model.Device{}

  _ = model.DB.First(&device, model.Device{Sn: entityId, ParentDin: dev.ParentDin})

  if (model.Device{}) == device {
    log.WithFields(log.Fields{
      "entityId": entityId,
      "gateWaySn": gateWaySn,
    }).Error("[MQTT] Device info is not exists!")
    return
  }

  click := 0

  if data.Type == "state_change" {
    click = 1
  }

  dType := strconv.Itoa(int(device.DType))

  token := GetToken(device.ParentDin)

  msg := ""

  online := data.Data["state"] != Offline
  attrs := data.Data["attributes"].(map[string]interface{})
  switch device.DType {
  case Light:
    isOn := data.Data["state"] == "on"
    // features 0 表示普通灯 1 表示调光灯
    //if (data.Data["state"] == "off" && attrs["supported_features"] == 1.0) {
    if attrs["brightness"] == nil{
      attrs["brightness"] = 0.0
    }
    status := [1]map[string]interface{}{}
    status[0] = map[string]interface{}{
      "button": 1,
      "on": isOn,
      "bright": attrs["brightness"].(float64),
    }
    attrs = map[string]interface{}{
      "datapointId": LightPointId,
      "click": click,
      "status": status,
    }
  case Switch:
    isOn := data.Data["state"] == "on"
    attrs = map[string]interface{}{
      "datapointId": SwitchPointId,
      "click": click,
      "switch": isOn,
    }
  case Cover:
    state := data.Data["state"]
    if state == "closed" {
      state = "close"
    }
    attrs = map[string]interface{}{
      "datapointId": CoverPointId,
      "click": click,
      "action": state,
    }
  case BinarySensor, PolyPirSensor:
    state := data.Data["state"] == "on"
    attrs = map[string]interface{}{
      "datapointId": BinarySensorPointId,
      "click": click,
      "sensor": state,
    }
  case PolyPanel4:
    attrs = map[string]interface{}{
      "datapointId": PolyPanel4PointId,
      "button": attrs["button"],
    }
  }

  jsonStr, _ := json.Marshal(attrs)
  msg = string(jsonStr)
  model.DB.Model(&device).Updates(map[string]interface{}{
    "state": data.Data["state"],
    "online": online,
    "attributes": msg,
  })

  MessageNotify(token, device.Din, dType, msg)
}
