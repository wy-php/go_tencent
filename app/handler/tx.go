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
)

const (
  BaseUrl = "https://api.jia.qq.com/iotd/"
)

func TxLogin() map[string]interface{} {
  log.WithFields(log.Fields{
    "appId": AppId,
    "appKey": AppKey,
  }).Info("[TXAPI] Call tx login api.")

  timestamp := time.Now().Unix() * 1000
  result := map[string]interface{}{
    "token": "",
    "expiryTime": 0,
  }
  sign := encrypt(AppKey, timestamp , RandNum)
  url := BaseUrl + "sp/login/?" + "spId=" + strconv.Itoa(AppId) + "&time=" + strconv.Itoa(int(timestamp)) + "&num=" + strconv.Itoa(RandNum) + "&sig=" + sign
  resp, err := http.Get(url)
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
      token := formatData(data, "token")
      result["token"] = token.(string)
      timeNum, _ := formatData(data, "expiryTime").(float64)
      result["expiryTime"] = int64(timeNum)
  }
  log.WithFields(log.Fields{
    "result": result,
  }).Info("[TXAPI] Call tx login api success!")
  return result
}

func TxDeviceRegister(token string, dType string, parentDin string, sn string, name string) string {

  log.WithFields(log.Fields{
    "token": token,
    "dType": dType,
    "parentDin": parentDin,
    "sn": sn,
    "name": name,
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
  jsonData.Set("parentDin", parentDin)
  jsonData.Set("sn", sn)
  jsonData.Set("name", name)
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

func TxDeviceUpdate(token string, dType string, parentDin string, sn string, name string, cDin string) string {
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
  jsonData.Set("sn", sn)
  jsonData.Set("name", name)
  response, err := http.PostForm(apiUrl,jsonData)
  if err != nil {
    log.WithFields(log.Fields{
      "err": err,
    }).Fatal("The HTTP request failed with error")
  } else {
    data, _ := ioutil.ReadAll(response.Body)
    din = formatData(data, "din").(string)
  }

  log.WithFields(log.Fields{
    "din": din,
  }).Info("[TXAPI] Call tx device update api success!")

  return din
}

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
  jsonData.Set("timestamp", strconv.Itoa(int(time.Now().Unix() * 1000)))
  response, err := http.PostForm(apiUrl,jsonData)

  if err != nil {
    log.WithFields(log.Fields{
      "err": err,
    }).Fatal("The HTTP request failed with error")
  } else {
    data, _ := ioutil.ReadAll(response.Body)
    log.WithFields(log.Fields{
      "data": string(data),
    }).Info("[TXAPI] Call tx message notify api success!")
  }

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
  gateWay := "601e5c44a4064d0c9b8e4ef49145ee10"
  din := TxDeviceRegister(token, "20002", "", gateWay, "智能网关")
  log.Info("din:", din)

  token = GetToken(din)
  lnlightDin := TxDeviceRegister(token, "20003", din, "light.lnlightb453", "智能开关")
  log.Info("lnlightDin:", lnlightDin)

  dimlightDin := TxDeviceRegister(token, "20003", din, "light.dimlight53a", "智能开关(调光灯)")
  log.Info("dimlightDin:", dimlightDin)

  socketDin := TxDeviceRegister(token, "20004", din, "switch.socket2b17", "智能插座")
  log.Info("socketDin:", socketDin)

  coverDin := TxDeviceRegister(token, "20007", din, "cover.cover32401", "智能窗帘")
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
  msg.Set("datapointId'", "1000204")
  msg.Set("switch", "false")
  msg.Set("metering", metMsg.Encode())
  MessageNotify(token, socketDin, "20004", msg.Encode())

  // 零火灯消息上报
  metMsg1 := url.Values{}
  metMsg.Set("totalEnergy", "100")
  metMsg.Set("voltage", "220")
  metMsg.Set("electricity", "0.5")
  metMsg.Set("power", "100")
  msg1 := url.Values{}
  msg.Set("datapointId'", "1000204")
  msg.Set("switch", "true")
  msg.Set("metering", metMsg1.Encode())

  MessageNotify(token, socketDin, "20003", msg1.Encode())

  cmd = exec.Command("sleep", "3")
  cmdErr = cmd.Start()

  // 更新设备
  din = TxDeviceUpdate(token, "20002", "", "polyhome-gateway-001", "智能网关1", din)
  log.Info("after update din:", din)

  if cmdErr != nil {
     log.Info(cmdErr)
   }

   log.Info("--------------------------------------------------------------")
   log.Info("call api end.")
 }
