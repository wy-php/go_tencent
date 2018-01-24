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
  r.ParseForm()

  log.WithFields(log.Fields{
    "Params": r.Form,
    "Content-type": r.Header.Get("Content-Type"),
  }).Info("Call spContoller api")

  num, _ := strconv.ParseInt(r.Form["num"][0], 10, 64)
  timestamp, _ := strconv.ParseUint(r.Form["timestamp"][0], 10, 64)
  dType, _ := strconv.ParseUint(r.Form["dType"][0], 10, 64)

  var cmd map[string]interface{}

  err := json.Unmarshal([]byte(r.Form["cmd"][0]), &cmd)


//  var cmd1 map[string]interface{}
//
//  err = json.Unmarshal([]byte(r.Form["cmd"][0]), &cmd1)

  result1 := new(model.Result)
  result1.Din = r.Form["din"][0]
  result1.Dtype = dType
  result1.ParentDin = r.Form["parentDin"][0]
  result1.Sn = r.Form["sn"][0]
  result1.Timestamp = timestamp
  result1.Num = num
  result1.Sig = r.Form["sig"][0]
  result1.Cmd = cmd

 // result2 := new(model.Result)
 // result2.Din = r.Form["din"][0]
 // result2.Dtype = dType
 // result2.ParentDin = r.Form["parentDin"][0]
 // result2.Sn = r.Form["sn"][0]
 // result2.Timestamp = timestamp
 // result2.Num = num
 // result2.Sig = r.Form["sig"][0]
 // result2.Cmd = cmd1

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

  sign := encrypt(AppKey, int64(result1.Timestamp), result1.Num)

  log.WithFields(log.Fields{
    "sign": sign,
    "old_sign": result1.Sig,
    "result1": result1,
  }).Info("-----------------sig----------------")

  if sign != result1.Sig {
    result["status"] = "ERROR"
    result["code"] = 400
    result["data"] = map[string]interface{}{
      "timestamp": time.Now().Unix(),
      "msg": "sig is error",
    }
  }else{
    // 整理传给主机所需要的数据格式
    reqParams := formatResult(result1)
    // 存储控制信息
    //saveControlInfo(result2)
    newReqParams, _ := json.Marshal(reqParams)

    device := model.Device{}

    _ = model.DB.First(&device, model.Device{Din: result1.ParentDin})

    mqttPubTopic := "/v1/polyhome-ha/host/"+ device.Sn + "/user_id/0/services/"

    model.MqttClient.Publish(mqttPubTopic, 0, false, newReqParams)
  }

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
