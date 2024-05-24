package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/miekg/dns"
	"log"
	"net"
	"strings"
	"time"
)

// 上游 DNS 服务器地址
const upstreamDNS = "114.114.114.114:53"

// forwardDNSRequest 转发 DNS 请求到上游 DNS 服务器
func forwardDNSRequest(w dns.ResponseWriter, r *dns.Msg, q dns.Question) {
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

// handleDNSRequest 处理传入的 DNS 请求
func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = true
	// 遍历所有问题并检查是否需要拦截
	for _, q := range r.Question {
		if q.Name == "io.cn"+"." || strings.HasSuffix(q.Name, ".io.cn"+".") {
			logs.Warning("Hijacking %s query for %s", dns.TypeToString[q.Qtype], q.Name)
			if q.Qtype == dns.TypeA {
				// 拦截特定域名并返回固定 IP
				a := &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    600,
					},
					A: net.ParseIP("0.0.0.0"),
				}
				msg.Answer = append(msg.Answer, a)
			} else if q.Qtype == dns.TypeAAAA {
				// 拦截特定域名并返回固定 IP
				a := &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
						Ttl:    600,
					},
					AAAA: net.ParseIP("240e:39e:3c3:47e0:69d:8986:7689:7eb4"),
				}
				msg.Answer = append(msg.Answer, a)
			} else if q.Qtype == dns.TypeHTTPS {
				// 拦截所有 *.example.com 的 HTTPS 记录请求并返回自定义响应
				httpsRecord := &dns.SVCB{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeHTTPS,
						Class:  dns.ClassINET,
						Ttl:    600,
					},
					Priority: 1,
					Target:   ".",
					Value: []dns.SVCBKeyValue{
						&dns.SVCBPort{
							Port: 443,
						},
						&dns.SVCBMandatory{
							Code: []dns.SVCBKey{443},
						},
					},
				}
				msg.Answer = append(msg.Answer, httpsRecord)
			} else {
				logs.Error(q.Qtype, q.Name)
				msg.Rcode = dns.RcodeNameError
				w.WriteMsg(&msg)
			}
		} else {
			// 转发其他查询到上游 DNS 服务器
			forwardDNSRequest(w, r, q)
		}
	}
	w.WriteMsg(&msg)
}

func main() {
	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)

	var handlerFunc = dns.HandlerFunc(handleDNSRequest)
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
	err := dns.ListenAndServe(":53", "tcp", handlerFunc)
	if err != nil {
		log.Fatalf("Failed to start TCP server: %v\n", err)
	}
}
