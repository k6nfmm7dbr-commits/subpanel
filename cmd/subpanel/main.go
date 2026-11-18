package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subpanel/internal/config"
	"subpanel/internal/server"
)

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func usage() {
	fmt.Fprintln(os.Stderr, `用法: subpanel [命令] [参数]

命令:
  serve                 启动面板服务（默认命令）
  version               显示版本
  config-get <key>      读取配置 (listen|data|admin_token)
  config-set <key> <值> 写入配置
  config-ensure-token   若无 admin_token 则自动生成并写入
  secret hex <字节数>   输出随机十六进制

参数 (serve):
  -conf    配置文件路径 (默认 $SUBPANEL_CONF 或 /etc/subpanel/config.json)
  -addr    监听地址 (默认 :20012)
  -data    节点数据文件
  -admin   管理令牌

环境变量:
  SUBPANEL_CONF, SUBPANEL_LISTEN, SUBPANEL_DATA, SUBPANEL_ADMIN`)
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Println("subpanel v" + config.Version)
			return
		case "secret":
			if len(os.Args) >= 4 && os.Args[2] == "hex" {
				var n int
				fmt.Sscan(os.Args[3], &n)
				if n <= 0 || n > 1024 {
					n = 32
				}
				fmt.Println(randomHex(n))
				return
			}
			usage()
			os.Exit(2)
		case "config-get", "config-set", "config-ensure-token":
			cmdConfig(os.Args[1], os.Args[2:])
			return
		case "serve":
			runServe(os.Args[2:])
			return
		case "-h", "--help", "help":
			usage()
			return
		}
	}
	// No subcommand: if first arg starts with a flag, treat as serve.
	if len(os.Args) > 1 && len(os.Args[1]) > 0 && os.Args[1][0] == '-' {
		runServe(os.Args[1:])
		return
	}
	usage()
	os.Exit(2)
}

func cmdConfig(cmd string, args []string) {
	path := config.DefaultPath()
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config load:", err)
		os.Exit(1)
	}
	switch cmd {
	case "config-get":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "usage: config-get <key>")
			os.Exit(2)
		}
		v, ok := cfg.Get(args[0])
		if !ok {
			fmt.Fprintln(os.Stderr, "unknown key:", args[0])
			os.Exit(2)
		}
		fmt.Println(v)
	case "config-set":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: config-set <key> <value>")
			os.Exit(2)
		}
		if !cfg.Set(args[0], args[1]) {
			fmt.Fprintln(os.Stderr, "unknown key:", args[0])
			os.Exit(2)
		}
		if err := config.Save(path, cfg); err != nil {
			fmt.Fprintln(os.Stderr, "config save:", err)
			os.Exit(1)
		}
		fmt.Println("ok")
	case "config-ensure-token":
		if cfg.AdminToken == "" {
			cfg.AdminToken = randomHex(24)
			if err := config.Save(path, cfg); err != nil {
				fmt.Fprintln(os.Stderr, "config save:", err)
				os.Exit(1)
			}
		}
		fmt.Println(cfg.AdminToken)
	}
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	def := config.Default()
	confPath := fs.String("conf", config.DefaultPath(), "config file path")
	addr := fs.String("addr", def.Listen, "listen address")
	data := fs.String("data", def.Data, "node data file")
	admin := fs.String("admin", "", "admin token")
	fs.Parse(args)

	cfg, err := config.Load(*confPath)
	if err != nil {
		log.Fatal("config load: ", err)
	}
	// Explicit CLI flags override config/env (unless they are flag defaults).
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "addr":
			cfg.Listen = *addr
		case "data":
			cfg.Data = *data
		case "admin":
			cfg.AdminToken = *admin
		}
	})
	if cfg.AdminToken == "" {
		log.Fatal("error: admin token required (-admin, SUBPANEL_ADMIN, or config admin_token)")
	}

	httpServer := &http.Server{
		Addr:              cfg.Listen,
		Handler:           nil, // set below
		ReadHeaderTimeout: 10 * time.Second,
	}

	// server.Serve builds its own handler; wrap it for graceful shutdown.
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down...")
		httpServer.Close()
	}()

	log.Printf("subpanel v%s listening on %s (data=%s)", config.Version, cfg.Listen, cfg.Data)
	if err := server.ServeHTTP(httpServer, cfg.Data, cfg.AdminToken); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
