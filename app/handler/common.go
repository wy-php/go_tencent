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
   Switch            = 20004
   PolyPanel4        = 20005
   BinarySensor      = 20006
   Cover             = 20007
   //PolyPirSensor     = 20008
   PolyIoSensor      = 20008
   LnLight3          = 20010
   LnLight           = 20011
   LnLight1          = 20012
   Sensor            = 20015
   Water             = 20016
   Lock              = 20017
   Light             = 20018
   PolySmokeSensor   = 20019
   PolyVapourSensor  = 20020
   LightPointId      = "1000202"
   SwitchPointId     = "1000204"
   CoverPointId      = "1000207"
   BinarySensorPointId     = "1000206"
   PolyPanel4PointId = "1000205"
   GatewayText       = "gateway"
   LightText         = "light"
   SwitchText        = "switch"
   LockText          = "lock"
   CoverText         = "cover"
   SensorText        = "sensor"
   WaterText         = "water"
   BinarySensorText  = "binary_sensor"
   PolyPirSensorText = "polypirsensor"
   PolyIoSensorText  = "polyiosensor"
   PolyPanel4Text    = "polypanel4"
   Offline           = "unavailable"
   PolySmokeSensorText = "polysmokesensor"
)

var Plugins = map[uint64]interface{}{
  Light: LightText,
  Switch: SwitchText,
  Cover: CoverText,
  Sensor: SensorText,
  BinarySensor: BinarySensorText,
  LnLight: LightText,
  LnLight1: LightText,
  LnLight3: LightText,
  PolyIoSensor: PolyIoSensorText,
  Lock: LockText,
  PolySmokeSensor: PolySmokeSensorText,
}

var Plugins1 = map[string]interface{}{
  LightText: Light,
  SwitchText: Switch,
  CoverText: Cover,
  GatewayText: Gateway,
  SensorText: Sensor,
  BinarySensorText: BinarySensor,
  //PolyPirSensorText: PolyPirSensor,
  PolyPanel4Text: PolyPanel4,
  PolyIoSensorText: PolyIoSensor,
  LockText: Lock,
  PolySmokeSensorText: PolySmokeSensor,
}

var loc, _ = time.LoadLocation("Asia/Shanghai")

func FindDtypeById(entityId string, deviceType string) int {
  prefix := strings.Split(entityId, ".")[1]
  dType := Plugins1[prefix]

  //如果是双键智能开关的话其dType就是20011
  if (prefix == LightText && deviceType == "polylnlight2") {
    dType = LnLight
  } else if (prefix == LightText && deviceType == "polylnlight3"){
    dType = LnLight3
  }else if (prefix == LightText && deviceType == "polylnlight"){
    dType = LnLight1
  }

  // io类又分了好几种
  if (prefix == BinarySensorText && deviceType == "polyiosensor") {
    dType = PolyIoSensor
  } else if (prefix == BinarySensorText && deviceType == "polysmokesensor"){
    dType = PolySmokeSensor
  }

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
  switch result.Dtype{
  case 20018:
    delete(result.Cmd, "button")
    if result.Cmd["on"] == true || result.Cmd["on"] == "true" {
      params["service"] = "turn_on"
      snStr := strings.SplitAfter(result.Sn, ".")[1]
      // 调光开关的。
      if strings.SplitAfter(snStr, "dimlight")[0] == "dimlight" {
        result.Cmd["brightness"] = result.Cmd["light"].(float64)*2.55
      }
    }
    delete(result.Cmd, "on")
    delete(result.Cmd, "light")

    params["data"] = result.Cmd
  case 20011:
    flag := result.Cmd["key"].(float64)
    uint64Flag := uint64(flag)


    if uint64Flag == 2 {
      result.Cmd["entity_id"] = result.Sn[0 : len(result.Sn)-1]+"2"
    }

    if result.Cmd["on"] == true {
      params["service"] = "turn_on"
    }
    delete(result.Cmd, "on")
    delete(result.Cmd, "key")

    params["data"] = result.Cmd
  case 20010:
    flag := result.Cmd["button"].(float64)
    uint64Flag := uint64(flag)

    if (uint64Flag == 2) {
      result.Cmd["entity_id"] = result.Sn[0 : len(result.Sn)-1]+"2"
    }else if(uint64Flag == 3) {
      result.Cmd["entity_id"] = result.Sn[0 : len(result.Sn)-1]+"3"
    }

    if result.Cmd["on"] == true {
      params["service"] = "turn_on"
    }
    delete(result.Cmd, "on")
    delete(result.Cmd, "button")

    params["data"] = result.Cmd
  case 20012:
    if result.Cmd["on"] == true {
      params["service"] = "turn_on"
    }
    delete(result.Cmd, "on")
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
      "ParentDin": dev.ParentDin,
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
    //brightness := attrs["brightness"].(float64)
    brightness := attrs["brightness"]
    if (brightness == nil){
      brightness = 0.0
    } else{
      brightness = brightness.(float64)/2.55
    }
    isOn := data.Data["state"] == "on"
    // features 0 表示普通灯 1 表示调光灯
    //if (data.Data["state"] == "off" && attrs["supported_features"] == 1.0) {
    //if attrs["brightness"] == nil{
    //  attrs["brightness"] = 0.0
    //}
    attrs = map[string]interface{}{
      //"datapointId": LightPointId,
      "click": click,
      "on": isOn,
      "lightUp": false,
      "lightDown": false,
      "light": brightness,
    }
  //双键智能开关
  case LnLight1:
    isOn := data.Data["state"] == "on"
    attrs = map[string]interface{}{
      "on": isOn,
    }
    //双键智能开关
  case LnLight:
    if (device.Sn == ""){
      log.Error("this data not find")
      return
    }
    light_location := device.Sn[len(device.Sn)-1 : len(device.Sn)]
    other_location := "1"
    if (light_location == "1"){
      other_location = "2"
    }
    prefix_info := device.Sn[0 : len(device.Sn)-1]
    //获取到另外的键的sn号
    other_light_sn := prefix_info + other_location
    //根据另外的那个键的sn号得到其状态信息
    other_light_info := model.Device{}
    _ = model.DB.First(&other_light_info, model.Device{Sn: other_light_sn, ParentDin: dev.ParentDin})

    if(other_light_info.State == ""){
      other_light_info.State = "off"
    }
    //定义一个数组包裹着集合的数据结构
    var g_employees = []interface{}{}

    on_status1 := data.Data["state"].(string)
    on_status2 := other_light_info.State
    if (light_location == "2"){
      on_status2 = data.Data["state"].(string)
      on_status1 = other_light_info.State
      device.Din = other_light_info.Din
    }

    status1 := false
    status2 := false
    if(on_status1 == "on"){
      status1 = true
    }
    if(on_status2 == "on"){
      status2 = true
    }
    map1 := map[string]interface{}{"key": 1, "on": status1}
    map2 := map[string]interface{}{"key": 2, "on": status2}

    g_employees = append(g_employees, map1)
    g_employees = append(g_employees, map2)

    //因为腾讯那里是双键智能开关也是只上报一个din的，所以这里就是只上传第一个din即可

    attrs = map[string]interface{}{
      "status": g_employees,
      "click": click,
    }
  case LnLight3:
    if (device.Sn == ""){
      log.Error("this data not find")
      return
    }
    light_location := device.Sn[len(device.Sn)-1 : len(device.Sn)]
    prefix_info := device.Sn[0 : len(device.Sn)-1]

    //获取到另外的键的sn号
    key1_sn := prefix_info + "1" //一键的
    key2_sn := prefix_info + "2" //二键的
    key3_sn := prefix_info + "3" //三键的
    key1_info := model.Device{}
    key2_info := model.Device{}
    key3_info := model.Device{}
    _ = model.DB.First(&key1_info, model.Device{Sn: key1_sn, ParentDin: dev.ParentDin})
    _ = model.DB.First(&key2_info, model.Device{Sn: key2_sn, ParentDin: dev.ParentDin})
    _ = model.DB.First(&key3_info, model.Device{Sn: key3_sn, ParentDin: dev.ParentDin})

    device.Din = key1_info.Din //上报到腾讯那里的din永远都是key1的din

    key1_status := key1_info.State
    key1_light := false
    if ( key1_status == "on"){
      key1_light = true
    }
    key2_status := key2_info.State
    key2_light := false
    if ( key2_status == "on"){
      key2_light = true
    }
    key3_status := key3_info.State
    key3_light := false
    if ( key3_status == "on"){
      key3_light = true
    }

    present_light := data.Data["state"].(string)
    present_light_status := false
    if (present_light == "on"){
      present_light_status = true
    }

    if (light_location == "1"){
      key1_light = present_light_status
    } else if(light_location == "2"){
      key2_light = present_light_status
    }else if(light_location == "3"){
      key3_light = present_light_status
    }

    //定义一个数组包裹着集合的数据结构
    var g_employees = []interface{}{}
    map1 := map[string]interface{}{"button": 1, "on": key1_light}
    map2 := map[string]interface{}{"button": 2, "on": key2_light}
    map3 := map[string]interface{}{"button": 3, "on": key3_light}

    g_employees = append(g_employees, map1)
    g_employees = append(g_employees, map2)
    g_employees = append(g_employees, map3)

    attrs = map[string]interface{}{
      "status": g_employees,
      "click": click,
    }
  case Switch:
    isOn := data.Data["state"] == "on"
    metering :=  map[string]interface{}{
      "totalEnergy": 0,
      "voltage": 0,
      "electricity": 0,
      "power": 0,
    }
    attrs = map[string]interface{}{
      //"datapointId": SwitchPointId,
      //"click": click,
      "metering": metering,
      "switch": isOn,
    }
  case Cover:
    state := data.Data["state"]
    if state == "closed" {
      state = "close"
    }
    attrs = map[string]interface{}{
      //"datapointId": CoverPointId,
      //"click": click,
      "action": state,
    }
  case Sensor:
    if (device.Sn == ""){
      log.Error("this data not find")
      return
    }
    light_location := device.Sn[len(device.Sn)-1 : len(device.Sn)]
    prefix_info := device.Sn[0 : len(device.Sn)-1]

    //获取到另外的键的sn号
    temperature_sn := prefix_info + "1" //温度的
    humidity_sn := prefix_info + "2" //湿度的
    light_sn := prefix_info + "3" //亮度的
    temperature := 0.0
    humidity := 0.0
    light := 0.0
    temperature_info := model.Device{}
    humidity_info := model.Device{}
    light_info := model.Device{}
    _ = model.DB.First(&temperature_info, model.Device{Sn: temperature_sn, ParentDin: dev.ParentDin})
    _ = model.DB.First(&humidity_info, model.Device{Sn: humidity_sn, ParentDin: dev.ParentDin})
    _ = model.DB.First(&light_info, model.Device{Sn: light_sn, ParentDin: dev.ParentDin})

    device.Din = temperature_info.Din //上报到腾讯那里的din永远都是温度的
    myMap := make(map[string]float64)
    temperature_content := temperature_info.Attributes
    if (temperature_content != ""){
      json.Unmarshal([]byte(temperature_content),&myMap)
      temperature = myMap["temperature"]
    }
    humidity_content := humidity_info.Attributes
    if (humidity_content != ""){
      json.Unmarshal([]byte(humidity_content),&myMap)
      humidity = myMap["humidity"]
    }
    light_content := light_info.Attributes
    if (light_content != ""){
      json.Unmarshal([]byte(light_content),&myMap)
      light = myMap["light"]
    }

    num_str := data.Data["state"].(string)
    num,err := strconv.ParseFloat(num_str, 64)
    if err != nil{
      log.Error(err)
    }
    if (light_location == "1"){
      temperature = num
    } else if(light_location == "2"){
      humidity = num
    }else if(light_location == "3"){
      light = num
    }

    attrs = map[string]interface{}{
      "temperature": temperature,
      "humidity": humidity,
      "light": light,
    }
  case BinarySensor:
    state := data.Data["state"] == "off"
    attrs = map[string]interface{}{
      //"datapointId": BinarySensorPointId,
      //"click": click,
      "low": false,
      "sensor": state,
    }
  case PolyIoSensor:
    state := data.Data["state"] == "on"
    attrs = map[string]interface{}{
      //"datapointId": BinarySensorPointId,
      //"click": click,
      "low": false,
      "sensor": state,
    }
  case Lock:
    door_state := ""
    if ( online == true){
      door_state = attrs["type"].(string)
    }
    state := data.Data["state"] == "unlocked"
    door_alarm := false
    low := false
    if (door_state == "trespass" || door_state == "prylock"){
      door_alarm = true
    } else if (door_state == "lowpower"){
      low = true
    }
    attrs = map[string]interface{}{
      //"datapointId": BinarySensorPointId,
      "on": state,
      "low": low,
      "lock_type": 0,
      "door_state": door_alarm,
    }
  case PolyPanel4:
    a := attrs["button"].(string)
    b,error := strconv.Atoi(a)
    if error != nil{
      fmt.Println("字符串转换成整数失败")
    }
    attrs = map[string]interface{}{
      "button": b,
    }
  case PolySmokeSensor:
    state := data.Data["state"] == "on"
    attrs = map[string]interface{}{
      "alarm": state,
      "low": false,
    }
  case Water:
    state := data.Data["state"] == "on"
    attrs = map[string]interface{}{
      "alarm": state,
      "low": false,
    }
  case PolyVapourSensor:
    state := data.Data["state"] == "on"
    attrs = map[string]interface{}{
      "alarm": state,
      "low": false,
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
