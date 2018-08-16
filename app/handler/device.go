package handler

import (
  "tda/app/model"
  "html/template"
  "encoding/json"
  "net/http"
  "strconv"
  "time"
  "strings"

  log "github.com/sirupsen/logrus"
  "fmt"
)

func SpController(w http.ResponseWriter, r *http.Request){
  //r.ParseForm()

  log.WithFields(log.Fields{
    "Params": r.Form,
    "Content-type": r.Header.Get("Content-Type"),
  }).Info("Call spContoller api")

  token := r.PostFormValue("token")
  sn_str := r.PostFormValue("sn")
  var list []string = strings.Split(sn_str,".")[1:]
  sn := strings.Join(list,".")
  din := r.PostFormValue("din")
  dType,_ := strconv.ParseUint(r.PostFormValue("dType"), 10, 64)
  cmds := r.PostFormValue("cmd")
  timestamp,_ := strconv.ParseUint(r.PostFormValue("timestamp"), 10, 64)
  parentDin := r.PostFormValue("parentDin")

  var cmd map[string]interface{}
  err := json.Unmarshal([]byte(cmds), &cmd)

  result1 := new(model.Result)
  result1.Token = token
  result1.Sn = sn
  result1.Din = din
  result1.Dtype = dType
  result1.Cmd = cmd
  result1.Timestamp = timestamp
  result1.ParentDin = parentDin

  result := map[string]interface{}{
    "code": 200,
    "status": "OK",
    "data": map[string]interface{}{
      "timestamp": time.Now().Unix(),
    },
  }


  w.Header().Set("Content-Type", "application/json")
  if err != nil {
    log.Error("json parse error", err)
    result["status"] = "ERROR"
    result["code"] = 400
    result["data"] = map[string]interface{}{
      "timestamp": time.Now().Unix(),
      "msg": "json parse error",
    }
  }

  // TODO 暂时不验证签名了
  //sign := encrypt(AppKey, int64(result1.Timestamp), result1.Num)
  //
  //log.WithFields(log.Fields{
  //  "sign": sign,
  //  "old_sign": result1.Sig,
  //  "result1": result1,
  //}).Info("-----------------sig----------------")
  //
  //if sign != result1.Sig {
  //  result["status"] = "ERROR"
  //  result["code"] = 400
  //  result["data"] = map[string]interface{}{
  //    "timestamp": time.Now().Unix(),
  //    "msg": "sig is error",
  //  }
  //}else{
  //  // 整理传给主机所需要的数据格式
  //  reqParams := formatResult(result1)
  //  // 存储控制信息
  //  //saveControlInfo(result2)
  //  newReqParams, _ := json.Marshal(reqParams)
  //
  //  device := model.Device{}
  //
  //  _ = model.DB.First(&device, model.Device{Din: result1.ParentDin})
  //
  //  mqttPubTopic := "/v1/polyhome-ha/host/"+ device.Sn + "/user_id/0/services/"
  //
  //  model.MqttClient.Publish(mqttPubTopic, 0, false, newReqParams)
  //}
  log.WithFields(log.Fields{
    "result": result1,
  }).Info("----------调试专用-------------")

  // 整理传给主机所需要的数据格式
  reqParams := formatResult(result1)
  // 存储控制信息
  //saveControlInfo(result2)
  newReqParams, _ := json.Marshal(reqParams)

  device := model.Device{}

  _ = model.DB.First(&device, model.Device{Din: result1.ParentDin})

  mqttPubTopic := "/v1/polyhome-ha/host/"+ device.Sn + "/user_id/0/services/"

  model.MqttClient.Publish(mqttPubTopic, 0, false, newReqParams)

  newResult, _ := json.Marshal(result)

  log.WithFields(log.Fields{
    "result": string(newResult),
    "topic": string(mqttPubTopic),
    "message": string(newReqParams),
  }).Info("return spController api result")

  w.Write(newResult)
}

func SpGetDeviceStatus(w http.ResponseWriter, r *http.Request) {
  params := r.URL.Query()

  num, _ := strconv.ParseInt(params["num"][0], 10, 64)

  timestamp, _ := strconv.ParseInt(params["timestamp"][0], 10, 64)

  oldSign := params["sig"][0]

  sign := encrypt(AppKey, timestamp, num)

  log.WithFields(log.Fields{
    "isSign": sign == oldSign,
    "sign": sign,
    "oldSign": oldSign,
    "Params": params,
  }).Info("Call SpGetDeviceStatus api")

  result := map[string]interface{}{}

  if sign != oldSign {
    result = map[string]interface{}{
      "code": 400,
      "status": "ERROR",
      "data": map[string]interface{}{
        "timestamp": time.Now().Unix(),
        "msg": "sig is error",
      },
    }
  }else{
    var ids []string
    ids = strings.Split(params["dins"][0], ",")

    type Res struct {
      Din string `json: "din"`
      Online bool `json: "isOnline"`
    }

    var res []Res
    model.DB.Table("devices").Select("din, online").Where("din in (?)", ids).Scan(&res)

    var arrs []map[string]interface{}
    for _, item  := range res {
      i := map[string]interface{}{}
      i["din"] = item.Din
      i["isOnline"] = item.Online
      arrs = append(arrs, i)
    }

    result = map[string]interface{}{
      "code": 200,
      "status": "OK",
      "data": arrs,
    }
  }

  newResult, _ := json.Marshal(result)

  log.WithFields(log.Fields{
    "result": string(newResult),
  }).Info("return spGetDeviceStatus api result")

  w.Header().Set("Content-Type", "application/json")

  w.Write(newResult)

}

func Index(w http.ResponseWriter, r *http.Request){
  type Todo struct {
    Task string
    Done bool
  }
  //template.ParseFiles("app/views/index.html")
  //todos := []Todo{
  //  {"Learn Go", true},
  //  {"Read Go Web Examples", true},
  //  {"Create a web app in Go", false},
  //}

  tmpl := template.Must(template.ParseFiles("app/views/index.html"))
  tmpl.Execute(w,  struct{}{})
}


func IndexPost(w http.ResponseWriter, r *http.Request){

  r.ParseForm()
  check_password := "polyhometencent"
  main_sn := r.PostFormValue("main_sn")
  device_sn := r.PostFormValue("device_sn")
  types := r.PostFormValue("type")
  password := r.PostFormValue("password")

  entityId := main_sn + "." + device_sn

  fmt.Println("main_sn is: ", main_sn)
  fmt.Println("device_sn is: ", device_sn)
  fmt.Println("types is: ", types)
  fmt.Println("password is: ", password)

  device := model.Device{}

  result := map[string]interface{}{
    "code": 200,
    "device_types": types,
  }

  _ = model.DB.First(&device, model.Device{Sn: entityId})

  log.WithFields(log.Fields{
    "device": device,
    "Sn": entityId,
    "ParentDin": main_sn,
  }).Info("数据库信息")

  //验证密码
  if (check_password != password){
    result = map[string]interface{}{
      "code": 201,
      "device_types": types,
    }
  }

  //数据库中存在性验证
  if (model.Device{}) == device {
    log.WithFields(log.Fields{
      "entityId": entityId,
      "ParentDin": main_sn,
    }).Error("[MQTT] Device info is not exists!")
    result = map[string]interface{}{
      "code": 202,
      "device_types": types,
    }
  }

  //如果数据存在并且dtyp又是一样的话，就不进行验证了。
  dType := strconv.Itoa(int(device.DType))
  if (dType == types){
    result = map[string]interface{}{
      "code": 203,
      "device_types": types,
    }
  }

  log.WithFields(log.Fields{
    "result": result,
  }).Info("返回的数据信息")

  //如果确定是正常的更新的话，就会更新本地数据库中的dtype字段，以及请求腾讯那里的接口更新腾讯那里的该字段。
  if (result["code"] == 200) {
    type_int,err:=strconv.Atoi(types)
    if err != nil {
      panic(err)
    }
    res := model.DB.Debug().Model(&device).Updates(map[string]interface{}{
      "d_type": type_int,
      //"updated_at": time.Now().Format("2006-01-02 15:04:05"),
    })

    if res.Error != nil {
      log.WithFields(log.Fields{
        "error_content": res.Error,
        "d_type": types,
        "device": device,
      }).Info("更新数据库错误")
    }

    token := GetToken(device.ParentDin)
    TxDeviceUpdate(token, types, device.ParentDin, device.Sn, device.Name, device.Din, "3")
  }

  //应答给请求。
  newResult, _ := json.Marshal(result)
  w.Write(newResult)
}
