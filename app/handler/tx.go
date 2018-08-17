package handler

import (
  "tda/app/model"

  "net/http"
  "net/url"
  "os/exec"
  "strconv"
  "time"
  "io/ioutil"

  log "github.com/sirupsen/logrus"
	"fmt"
)

const (
  BaseUrl = "https://api.jia.qq.com/iotd/"
)

//通过appkey获取sig
func TxgGetSig() map[string]interface{} {
  log.WithFields(log.Fields{
    "appId": AppId,
    "appKey": AppKey,
  }).Info("[TXAPI] Call tx getSig api.")

  result := map[string]interface{}{
    "random": 0,
    "sig": "",
  }
  getUrl := BaseUrl + "sp/getLoginParam?" + "skey=" + AppKey
  res, err := http.Get(getUrl)

  defer res.Body.Close()

  if err != nil {
    log.WithFields(log.Fields{
      "err": err,
    }).Fatal("[TXAPI] The HTTP request failed with get sig")
  } else {
    data, err :=ioutil.ReadAll(res.Body)
    if err != nil {
      log.Error("[TXAPI] tx get sig api error!")
    }
    if res.StatusCode < 200 || res.StatusCode > 299 {
      log.Error("[TXAPI] api server error")
      return result
    }
    random := formatData(data, "random").(float64)
    result["random"] = int64(random)
    timeNum := formatData(data, "sig")
    result["sig"] = timeNum.(string)
  }
  return result
}

//用来获取token的
func TxLogin() map[string]interface{} {
  getData := TxgGetSig()
  random := strconv.FormatInt(getData["random"].(int64), 10)
  sig := getData["sig"].(string)
  timestamp := time.Now().Unix() * 1000
  result := map[string]interface{}{
    "token": "",
    "expiryTime": 0,
  }

  apiUrl := BaseUrl + "sp/login"
  jsonData := url.Values{}
  jsonData.Set("appId", strconv.Itoa(AppId))
  jsonData.Set("time", strconv.Itoa(int(timestamp)))
  jsonData.Set("num", random)
  jsonData.Set("sig", sig)

  resp, err := http.PostForm(apiUrl, jsonData)
  if err != nil {
    log.WithFields(log.Fields{
      "err": err,
    }).Fatal("[TXAPI] The HTTP request failed with error")
  } else {
      data, err := ioutil.ReadAll(resp.Body)
      if err != nil {
        log.Error("[TXAPI] tx login api error!")
      }
      if resp.StatusCode < 200 || resp.StatusCode > 299 {
        log.Error("[TXAPI] api server error")
        return result
      }
	  fmt.Printf("%s", data)
      token := formatData(data, "token")
      if token == nil{
        log.Error("[TXAPI] token get error")
        return result
      }
      result["token"] = token.(string)
      timeNum, _ := formatData(data, "expiryTime").(float64)
      result["expiryTime"] = int64(timeNum)
  }
  log.WithFields(log.Fields{
    "result": result,
  }).Info("[TXAPI] Call tx login api success!")
  return result
}

//用来注册设备的
func TxDeviceRegister(token string, dType string, parentDin string, sn string, name string, types string) string {

  log.WithFields(log.Fields{
    "token": token,
    "dType": dType,
    "parentDin": parentDin,
    "sn": sn,
    "name": name,
    "type": types,
  }).Info("[TXAPI] Call tx device register api.")

  // check is register
  deviceInfo := model.Device{}
  model.DB.First(&deviceInfo, model.Device{Sn: sn, ParentDin: parentDin})
  if (model.Device{}) != deviceInfo  {
    log.WithFields(log.Fields{
      "din": deviceInfo.Din,
    }).Warn("[TXAPI] The device already register!")
    return deviceInfo.Din
  }



  apiUrl := BaseUrl + "sp/deviceRegister/"
  din := ""
  jsonData := url.Values{}
  jsonData.Set("token", token)
  jsonData.Set("dType", dType)
  jsonData.Set("type", types)
  jsonData.Set("parentDin", parentDin)
  jsonData.Set("sn", sn)
  jsonData.Set("name", name)

  //因为和腾讯那里的协议的原因，这里需要调整双键智能开关的逻辑
  new_sn := ""
  if (dType == "20011" || dType == "20015" || dType == "20010"){
    num := len(sn)
    location_num := sn[num-1 : num]
    prefix_info := sn[0 : num-1]
    //如果后缀是2的话，也把其改成1的。因为腾讯那里协议的原因，双键智能开关只接受一个din的注册
    if (location_num == "2"){
      new_sn = prefix_info+"1"
      jsonData.Set("sn", new_sn)
    }else if (location_num == "3"){
      new_sn = prefix_info+"1"
      jsonData.Set("sn", new_sn)
    }
  } else if (dType == "20008") {
    name = "红外人体感应"
  }

  log.WithFields(log.Fields{
    "token": token,
    "dType": dType,
    "parentDin": parentDin,
    "sn": sn,
    "name": name,
    "type": types,
  }).Info("--------------调试专用---------------")


  response, err := http.PostForm(apiUrl,jsonData)
  if err != nil {
    log.WithFields(log.Fields{
      "err": err,
    }).Fatal("[TXAPI] The HTTP request failed with error")
  } else {
    data, err := ioutil.ReadAll(response.Body)
    din = formatData(data, "din").(string)
    if err != nil {
      log.Error("[TXAPI] Call register api error: ", err)
      return din
    }else if din == "" {
      log.Error("[TXAPI] Call register api din error: ", string(data))
      return din
    }
 }

  // save device info
  if parentDin == "" {
    parentDin = din
  }

  newDType, _ := strconv.ParseUint(dType, 10, 64)

  device := model.Device{
    Din: din,
    DType: newDType,
    ParentDin: parentDin,
    Sn: sn,
    Name: name,
  }

  model.DB.Where(device).FirstOrCreate(&device)

  if err := model.DB.Save(&device).Error; err != nil {
    log.WithFields(log.Fields{
      "device": device,
    }).Error("[TXAPI] Save device to mysql error")
  } else {
    log.WithFields(log.Fields{
      "device": device,
    }).Info("[TXAPI] Save device to mysql success")
  }

  log.WithFields(log.Fields{
    "din": din,
  }).Info("[TXAPI] Call tx device register api success!")

  return din
}

//用来更新设备信息的
func TxDeviceUpdate(token string, dType string, parentDin string, sn string, name string, cDin string, types string) string {
  log.WithFields(log.Fields{
    "token": token,
  }).Info("[TXAPI] Call tx device update api.")

  apiUrl := BaseUrl + "sp/deviceUpdate/"
  din := ""
  jsonData := url.Values{}
  jsonData.Set("token", token)
  jsonData.Set("dType", dType)
  jsonData.Set("parentDin", parentDin)
  jsonData.Set("din", cDin)
  jsonData.Set("type", types)
  jsonData.Set("sn", sn)
  jsonData.Set("name", name)
  response, err := http.PostForm(apiUrl,jsonData)
  if err != nil {
    log.WithFields(log.Fields{
      "err": err,
    }).Fatal("The HTTP request failed with error")
  } else {
    data, _ := ioutil.ReadAll(response.Body)
    fmt.Printf("%s", data)
    din = formatData(data, "din").(string)
  }

  log.WithFields(log.Fields{
    "din": din,
  }).Info("[TXAPI] Call tx device update api success!")

  return din
}

//用来上报消息到腾讯接口
func MessageNotify(token string, din string, dType string, msg string) {

  log.WithFields(log.Fields{
    "token": token,
    "din": din,
    "dType": dType,
    "msg": msg,
  }).Info("[TXAPI] Call tx message notify api.")

  apiUrl := BaseUrl + "message/notify/"
  jsonData := url.Values{}
  jsonData.Set("token", token)
  jsonData.Set("din", din)
  jsonData.Set("dType", dType)
  jsonData.Set("msg", msg)
  jsonData.Set("timestamp", strconv.Itoa(int(time.Now().Unix())))
  response, err := http.PostForm(apiUrl,jsonData)

  if err != nil {
    log.WithFields(log.Fields{
      "err": err,
    }).Fatal("The HTTP request failed with error")
  } else {
    data, _ := ioutil.ReadAll(response.Body)
    fmt.Printf("%s", data)
    log.WithFields(log.Fields{
      "data": string(data),
    }).Info("[TXAPI] Call tx message notify api success!")
  }

}

//注册控制设备请求的时候的uri
func TxRegisterUri() map[string]interface{}{
  result := map[string]interface{}{
    "appId": "",
    "appKey": "",
    "msg": "fail",
  }

  apiUrl := BaseUrl + "sp/register"
  jsonData := url.Values{}
  jsonData.Set("spAppId", SpAppId)
  jsonData.Set("spSkey", SpSkey)
  jsonData.Set("name", Name)
  jsonData.Set("desc", Desc)
  jsonData.Set("spUri", Url)

  resp, err := http.PostForm(apiUrl, jsonData)
  if err != nil {
    log.WithFields(log.Fields{
      "err": err,
    }).Fatal("[TXAPI] The HTTP request failed with error")
  } else {
    data, err := ioutil.ReadAll(resp.Body)
    if err != nil {
      log.Error("[TXAPI] tx register api error!")
    }
    if resp.StatusCode < 200 || resp.StatusCode > 299 {
      log.Error("[TXAPI] api server error")
      return result
    }
    result["appId"] = formatData(data, "appId")
    result["appKey"] = formatData(data, "skey")
    result["msg"] = "success"
  }
  log.WithFields(log.Fields{
    "result": result,
  }).Info("[TXAPI] Call tx login api success!")
  return result
}

func CallTxApi(){
  log.Info("start call tx api...")
  log.Info("--------------------------------------------------------------")

  // 获取token
  token := TxLogin()["token"].(string)
  //  token := "b1d0bf0f3dc45cf8c116f74e6c17c3a3"
  log.Info("token:", token)

  cmd := exec.Command("sleep", "1")
  cmdErr := cmd.Start()

  // 注册设备
  //gateWay := "dc:a9:04:99:09:64"
  gateWay := "601e5c44a4064d0c9b8e4ef49145ee11"
  din := TxDeviceRegister(token, "20002", "", gateWay, "智能网关", "2")
  log.Info("din:", din)

  token = GetToken(din)
  lnlightDin := TxDeviceRegister(token, "20003", din, "light.lnlightb453", "智能开关", "3")
  log.Info("lnlightDin:", lnlightDin)

  dimlightDin := TxDeviceRegister(token, "20003", din, "light.dimlight53a", "智能开关(调光灯)", "3")
  log.Info("dimlightDin:", dimlightDin)

  socketDin := TxDeviceRegister(token, "20004", din, "switch.socket2b17", "智能插座", "3")
  log.Info("socketDin:", socketDin)

  coverDin := TxDeviceRegister(token, "20007", din, "cover.cover32401", "智能窗帘", "3")
  log.Info("coverDin:", coverDin)

  cmd = exec.Command("sleep", "1")
  cmdErr = cmd.Start()

  // 插座消息上报
  metMsg := url.Values{}
  metMsg.Set("totalEnergy", "100")
  metMsg.Set("voltage", "220")
  metMsg.Set("electricity", "0.5")
  metMsg.Set("power", "100")
  msg := url.Values{}
  msg.Set("switch", "true")
  msg.Set("metering", metMsg.Encode())
  MessageNotify(token, socketDin, "20004", msg.Encode())

  // 零火灯消息上报
  metMsg1 := url.Values{}
  metMsg.Set("totalEnergy", "100")
  metMsg.Set("voltage", "220")
  metMsg.Set("electricity", "0.5")
  metMsg.Set("power", "100")
  msg1 := url.Values{}
  msg.Set("switch", "true")
  msg.Set("metering", metMsg1.Encode())

  MessageNotify(token, socketDin, "20003", msg1.Encode())

  cmd = exec.Command("sleep", "3")
  cmdErr = cmd.Start()

  // 更新设备
  din = TxDeviceUpdate(token, "20002", "1000210042", "polyhome-gateway-001", "智能网关2", din, "2")
  log.Info("after update din:", din)

  if cmdErr != nil {
     log.Info(cmdErr)
   }

   log.Info("--------------------------------------------------------------")
   log.Info("call api end.")
 }
