package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	domainFile string
	beianDomainFile string
	unbeianDomainFile string
	domainDb string
	searchUrl string
	con int

	bal map[byte] *node = make(map[byte] *node)
	ubal map[byte] *node = make(map[byte] *node)
	balF *os.File
	ubalF *os.File
	conchan chan bool
	mutex sync.Mutex
	mutex2 sync.Mutex
)

func init()  {
	flag.StringVar(&domainFile, "d", "./list.txt","等待检查的域名列表文件")
	flag.StringVar(&domainDb, "db", "./domain.db","存储bal.txt和ubal.txt的数据库文件")
	flag.StringVar(&beianDomainFile, "b", "./bal.txt","已查备案的域名列表文件")
	flag.StringVar(&unbeianDomainFile, "ub", "./ubal.txt","已查未备案的域名列表文件")
	flag.StringVar(&searchUrl, "su", "","查询服务器地址")
	flag.IntVar(&con, "c", 1,"查询并发数")
	flag.Parse()
}
func main()  {
	Init()

	f, err := OpenFile(domainFile)
	if err != nil {
		unInit()
		log.Panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.Trim(line, "\n")
		line = strings.Trim(line, "\r")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		conchan <- true
		go checkBeian(line)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool)
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-sigCh:
				done<-true
				return
			case <-ticker.C:
				l := len(conchan)
				if l == 0 {
					done<-true
					return
				}
			}
		}
	}()
	<- done
	unInit()

}

func checkBeian(domain string)  {
	defer func() {
		<- conchan
	}()
	if hasDomain(domain, bal) {
		log.Printf(domain + " 已查备案了")
		return
	}
	if hasDomain(domain, ubal) {
		log.Printf(domain + " 已查未备案")
		return
	}
	var resp *http.Response
	var err error
	var url string
	for retry := 3; retry > 0; retry-- {
		url = searchUrl + domain
		resp, err = http.Get(url)
		if err == nil {
			break
		} else {
			log.Printf(url)
			log.Printf(err.Error())
		}
	}
	if err != nil {
		return
	}
	defer resp.Body.Close()

	byets, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf(err.Error())
		return
	}
	var info = &struct {
		Code int
	}{}
	err = json.Unmarshal(byets, info)
	if err != nil {
		log.Printf(string(byets))
		log.Printf(err.Error())
		return
	}
	code := info.Code
	if code == 1 {
		fillInDomain(domain, bal)
		writeDomain(domain, balF)
		log.Printf(domain + " 备案了")
	} else if code == 0 {
		fillInDomain(domain, ubal)
		writeDomain(domain, ubalF)
		log.Printf(domain +  "没备案")
	} else {
		log.Printf(domain + " code " + strconv.Itoa(code))
	}
}

func unInit()  {

	if balF != nil {
		balF.Close()
	}
	if ubalF != nil {
		ubalF.Close()
	}
	log.Printf("program exit")
}

func writeDomain(domain string, f *os.File)  {
	mutex.Lock()
	defer mutex.Unlock()
	if _, err := io.WriteString(f, domain +"\n"); err != nil {
		log.Printf("dsssssss")
		log.Fatal(err.Error())
	}
}
func Init()  {
	var err error
	balF, err = os.OpenFile(beianDomainFile, os.O_APPEND|os.O_RDWR, os.ModePerm)
	if err != nil {
		log.Panic(err)
	}
	scanner := bufio.NewScanner(balF)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.Trim(line, "\n")
		line = strings.Trim(line, "\r")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fillInDomain(line, bal)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	ubalF, err = os.OpenFile(unbeianDomainFile, os.O_APPEND|os.O_RDWR, os.ModePerm)
	if err != nil {
		balF.Close()
		log.Panic(err)
	}
	scanner = bufio.NewScanner(ubalF)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.Trim(line, "\n")
		line = strings.Trim(line, "\r")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fillInDomain(line, ubal)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	conchan = make(chan bool, con)

}

type node struct{
	end bool
	folw map[byte] *node
}

func reverse(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}
func hasDomain(domain string, m map[byte] *node) (has bool) {
	mutex2.Lock()
	defer mutex2.Unlock()
	bytes := []byte(domain)
	reverse(bytes)
	var preNode *node = nil
	var ok bool = false
	has = false
	for _, b := range bytes {
		if preNode == nil {
			preNode, ok = m[b]
			if !ok {
				has = false
				return
			}
		} else {
			preNode2, ok := preNode.folw[b]
			if !ok {
				has = false
				return
			}
			preNode = preNode2
		}

	}
	if preNode != nil {
		if preNode.end {
			has = true
			return
		}
	}
	return
}

func fillInDomain(domain string, m map[byte] *node) {
	mutex2.Lock()
	defer mutex2.Unlock()
	bytes := []byte(domain)
	reverse(bytes)
	var preNode *node = nil
	var ok bool = false
	for _, b := range bytes {
		if preNode == nil {
			preNode, ok = m[b]
			if !ok {
				preNode = &node{
					end:  false,
					folw: make(map[byte] *node),
				}
				m[b] = preNode
			}
		} else {
			preNode2, ok := preNode.folw[b]
			if !ok {
				preNode2 = &node{
					end:  false,
					folw: make(map[byte] *node),
				}
				preNode.folw[b]= preNode2
			}
			preNode = preNode2
		}

	}
	if preNode != nil {
		if !preNode.end {
			preNode.end = true
		}
	}
}

func OpenFile(filename string) (*os.File, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return os.Create(filename)
	}
	return os.OpenFile(filename, os.O_APPEND|os.O_RDWR, os.ModePerm)
}