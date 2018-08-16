package main

import (
  "tda/app"
  _ "tda/app/model"
  _ "fmt"
  _ "encoding/json"
  _ "tda/app/handler"
)

func main() {
  app := &app.App{}
  app.Initialize()

  // 调用腾讯接口
  //handler.TxRegisterUri()
  //handler.CallTxApi()
  app.Run(":9090")
}
