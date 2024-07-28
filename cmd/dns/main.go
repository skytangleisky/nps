package main

import (
	"database/sql"
	"fmt"
	"github.com/astaxie/beego/logs"
	_ "github.com/go-sql-driver/mysql"
	"github.com/miekg/dns"
	"log"
	"net"
	"reflect"
	"strings"
	"time"
)

type Record struct {
	Id         int64
	Uuid       string
	Domain     string
	Name       string
	Type       string
	Isp        string
	Record     string
	TTL        int64
	Status     string
	Remark     string
	Createtime string
}

// 上游 DNS 服务器地址
const upstreamDNS = "114.114.114.114:53"

type DnsServer struct {
	db *sql.DB
}

// forwardDNSRequest 转发 DNS 请求到上游 DNS 服务器
func (s *DnsServer) forwardDNSRequest(w dns.ResponseWriter, r *dns.Msg, q dns.Question) {
	client := new(dns.Client)
	if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		client.Net = "tcp"
	} else {
		client.Net = "udp"
	}
	client.Timeout = 5 * time.Second
	response, _, err := client.Exchange(r, upstreamDNS)
	if err != nil {
		logs.Error("Failed to forward %s query for %s: %v", client.Net, q.Name, err)
		return
	}
	logs.Info("Forward %s query for %s", client.Net, q.Name)
	w.WriteMsg(response)
}

func (s *DnsServer) mapToStruct(data map[string]interface{}, result interface{}) interface{} {
	r := reflect.ValueOf(result).Elem()
	for k, v := range data {
		f := r.FieldByName(k)
		if f.IsValid() && f.CanSet() {
			f.Set(reflect.ValueOf(v).Convert(f.Type()))
		}
	}
	return result
}

func (s *DnsServer) process(q dns.Question, answerPtr *[]dns.RR, record Record) {
	if record.Type == "A" {
		*answerPtr = append(*answerPtr, &dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    uint32(record.TTL),
			},
			A: net.ParseIP(record.Record),
		})
		logs.Warning("Hijacking %s query for %s", dns.TypeToString[dns.TypeA], record.Name)
	} else if record.Type == "AAAA" {
		*answerPtr = append(*answerPtr, &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    uint32(record.TTL),
			},
			AAAA: net.ParseIP(record.Record),
		})
		logs.Warning("Hijacking %s query for %s", dns.TypeToString[dns.TypeAAAA], record.Name)
	}
}

// handleDNSRequest 处理传入的 DNS 请求
func (s *DnsServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = true
	// 遍历所有问题并检查是否需要拦截
	for _, q := range r.Question {
		results := s.getResult()
		flag := false
		for _, result := range results {
			record := Record{}
			s.mapToStruct(result, &record)
			if record.Status == "启用" {
				if record.Domain == "*" {
					if strings.HasSuffix(q.Name, "."+record.Name+".") {
						s.process(q, &msg.Answer, record)
						flag = true
					}
				} else if record.Domain == "@" {
					if q.Name == record.Name+"." {
						s.process(q, &msg.Answer, record)
						flag = true
					}
				} else {
					if q.Name == record.Domain+"."+record.Name+"." {
						s.process(q, &msg.Answer, record)
						flag = true
					}
				}
			}
		}
		if flag == false {
			s.forwardDNSRequest(w, r, q)
		}
	}
	w.WriteMsg(&msg)
}

func (s *DnsServer) getResult() []map[string]interface{} {
	beginTime := time.Now()
	defer func() {
		endTime := time.Now()
		elapsed := endTime.Sub(beginTime)
		fmt.Println(elapsed)
	}()
	// 查询数据
	rows, err := s.db.Query("SELECT * FROM `dns`")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		panic(err)
	}
	// 创建一个切片来存储列值
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}
	var results []map[string]interface{}
	// 处理查询结果
	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			panic(err)
		}
		// 打印结果
		result := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				result[col] = string(b)
			} else {
				result[col] = val
			}
		}
		results = append(results, result)
	}
	// 检查是否有错误
	err = rows.Err()
	if err != nil {
		panic(err)
	}
	return results
}

func main() {
	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	db, err := sql.Open("mysql", "admin:tanglei@tcp(123.57.209.17:3306)/union")
	//db, err := sql.Open("mysql", "root:tanglei@tcp(192.168.101.104:3306)/union")
	//db, err := sql.Open("mysql", "root:tanglei@tcp(127.0.0.1:3306)/union")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	// 测试数据库连接
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("成功连接到数据库！")
	s := &DnsServer{db}
	var handlerFunc = dns.HandlerFunc(s.handleDNSRequest)
	// 启动 UDP DNS 服务器
	go func() {
		log.Printf("Starting UDP DNS server on port 53")
		err := dns.ListenAndServe(":53", "udp", handlerFunc)
		if err != nil {
			log.Fatalf("Failed to start UDP server: %v\n", err)
		}
	}()

	// 启动 TCP DNS 服务器
	log.Printf("Starting TCP DNS server on port 53")
	err = dns.ListenAndServe(":53", "tcp", handlerFunc)
	if err != nil {
		log.Fatalf("Failed to start TCP server: %v\n", err)
	}
}
