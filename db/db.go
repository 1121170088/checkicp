package db

import (
	"bufio"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strings"
)

var (
	db *sql.DB
)
func Init(dbFile string, ubaltxt string, baltxt string)  {
	var init = false
	fi , err := os.Stat(dbFile)
	if err != nil {
		init = true
	} else {
		log.Printf(fi.Name())
	}
	db, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	if init {
		log.Printf("init db ...")
		sqlStmt := `
create table bal (domain text null primary key);
create table ubal (domain text null primary key);
`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			db.Close()
			log.Fatal(err)
		}
		balF, err := os.OpenFile(baltxt, os.O_APPEND|os.O_RDWR, os.ModePerm)
		if err != nil {
			db.Close()
			log.Panic(err)
		}
		defer balF.Close()
		scanner := bufio.NewScanner(balF)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.Trim(line, "\n")
			line = strings.Trim(line, "\r")
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			InsertBal(line)
		}
		if err := scanner.Err(); err != nil {
			db.Close()
			log.Fatal(err)
		}
		ubalF, err := os.OpenFile(ubaltxt, os.O_APPEND|os.O_RDWR, os.ModePerm)
		if err != nil {
			balF.Close()
			db.Close()
			log.Panic(err)
		}
		defer ubalF.Close()
		scanner = bufio.NewScanner(ubalF)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.Trim(line, "\n")
			line = strings.Trim(line, "\r")
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			InsertUbal(line)
		}
		if err := scanner.Err(); err != nil {
			balF.Close()
			ubalF.Close()
			db.Close()
			log.Fatal(err)
		}
	}
}

func SelectBal(domain string) bool {
	stmt, err := db.Prepare("select domain from bal where domain=?")
	if err != nil {
		log.Printf(err.Error())
		return false
	}
	defer stmt.Close()
	if stmt.QueryRow(domain).Scan()==sql.ErrNoRows {
		return false
	} else {
		return true
	}
}

func InsertBal(domain string) {
	stmt, err := db.Prepare("insert into bal(domain) values (?)")
	if err != nil {
		log.Printf(err.Error())
		return
	}
	defer stmt.Close()
	stmt.Exec(domain)
}

func SelectUbal(domain string) bool {
	stmt, err := db.Prepare("select domain from ubal where domain=?")
	if err != nil {
		log.Printf(err.Error())
		return false
	}
	defer stmt.Close()
	if stmt.QueryRow(domain).Scan()==sql.ErrNoRows {
		return false
	} else {
		return true
	}
}

func InsertUbal(domain string) {
	stmt, err := db.Prepare("insert into ubal(domain) values (?)")
	if err != nil {
		log.Printf(err.Error())
		return
	}
	defer stmt.Close()
	stmt.Exec(domain)
}

func Unint()  {
	db.Close()
}