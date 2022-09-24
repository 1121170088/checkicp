package main

import (
	"checkicp/db"
	"log"
	"testing"
)

func Test_Db(t *testing.T)  {
	db.Init("domain.db")
	log.Printf("%v", db.SelectBal("zhansan.com"))
	db.InsertBal("zhansan.com")
	log.Printf("%v", db.SelectBal("zhansan.com"))
	db.Unint()
}
