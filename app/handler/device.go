package handler

import (
  "tda/app/model"
  "encoding/json"
  "net/http"
  "strconv"
  "time"
  "strings"

  log "github.com/sirupsen/logrus"
)

func SpController(w http.ResponseWriter, r *http.Request){
  //r.ParseForm()

  log.WithFields(log.Fields{
    "Params": r.Form,
    "Content-type": r.Header.Get("Content-Type"),
  }).Info("Call spContoller api")

  token := r.PostFormValue("token")
  sn := r.PostFormValue("sn")
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
