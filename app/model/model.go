package model

import (
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/mysql"

)

type Device struct {
  gorm.Model
  Din         string `gorm:"not null;unique" json: "din"`
  DType       uint64 `json: "dType"`
  ParentDin   string `json: "parentDin"`
  Sn          string `json: "sn"`
  Name        string `json: "name"`
  State       string `json: "state"`
  Online      bool
  Attributes  string `json: "attributes"`
}

type Payload struct {
  Type      string `json: "type"`
  Status    string `json: "status"`
  Data      map[string]interface{} `json: "data"`
    //var a [1]map[string]interface{}
  //Data      [99]map[string]interface{} `json: "data"`
}

type ReportPayload struct {
  Type      string `json: "type"`
  Status    string `json: "status"`
  Data      []State
}

type State struct {
  EntityId    string `json: "entityId"`
  State     string `json: "state"`
}

type Result struct {
  Token      string `json: "token"`
  Din        string `json: "din"`
  Dtype      uint64 `json: "deviceType"`
  //Plugin     string `json:"plugin"`
  ParentDin  string `json: "parentDin"`
  Sn         string `json:"sn"`
  Timestamp  uint64 `json:"timestamp"`
  Cmd        map[string]interface{} `json:"cmd"`
}

func DBMigrate(db *gorm.DB) *gorm.DB {
  db.AutoMigrate(&Device{})
  return db
}
