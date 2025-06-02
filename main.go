package main

import (
	"log"
	"net"
	"net/http"
	"os/exec"

	"github.com/HCH1212/utils/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// 中间件：仅允许局域网访问
	r.Use(localNetworkOnly())
	r.Use(middleware.CorsGin())

	// 路由
	r.POST("/file", uploadfile)
	r.GET("/file", downloadfile)
	r.DELETE("/file", closefile)
	r.GET("/list", listfiles)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// 前端
	r.GET("/", func(c *gin.Context) {
		c.File("index.html")
	})

	// 获取本地局域网 IP
	localIP := getLocalIP()
	if localIP == "" {
		log.Fatal("无法获取局域网 IP")
	}

	addr := localIP + ":8088"
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

// 中间件：限制仅允许局域网 IP 访问
func localNetworkOnly() gin.HandlerFunc {
	privateCIDRs := []string{
		"192.168.0.0/16",
		"10.0.0.0/8",
		"172.16.0.0/12",
	}

	var ipNets []*net.IPNet
	for _, cidr := range privateCIDRs {
		_, ipNet, _ := net.ParseCIDR(cidr)
		ipNets = append(ipNets, ipNet)
	}

	return func(c *gin.Context) {
		// 先ping一下对面，因为网校局域网的问题
		_ = pingIP(c.ClientIP())

		clientIP := net.ParseIP(c.ClientIP())
		allowed := false
		for _, ipNet := range ipNets {
			if ipNet.Contains(clientIP) {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied. Only local network access is allowed."})
			c.Abort()
			return
		}
		c.Next()
	}
}

// 获取局域网 IP
func getLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("获取网络接口失败: %v", err)
		return ""
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
				return ip.String()
			}
		}
	}
	return ""
}

// pingIP 用于 ping 指定的 IP 地址
func pingIP(ip string) error {
	// 调用系统 ping 命令，发送 1 个 ICMP 请求
	cmd := exec.Command("ping", "-c", "1", "-w", "1", ip)
	err := cmd.Start() // 启动命令
	if err != nil {
		return err
	}

	// 不等待完整输出，立即返回
	return nil
}
