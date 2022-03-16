package main

import (
	"embed"
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"nginx-proxy-manager/app/config"
	"nginx-proxy-manager/app/middlewares"
	"nginx-proxy-manager/app/routes"
	"nginx-proxy-manager/app/services"
	"os"
	"syscall"
)

//go:embed views
var templates embed.FS

//go:embed views/public
var staticFS embed.FS

func mustFS() http.FileSystem {
	sub, err := fs.Sub(staticFS, "views/public")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}

func main() {
	// 线上模式显示版本信息
	if gin.Mode() == gin.ReleaseMode {
		config.DisplayVersion()
	}
	// 初始化配置文件
	config.InitAppConfig()
	// 初始化redis
	config.InitRedis()
	defer config.CloseRedis()

	app := gin.New()
	template, _ := template.ParseFS(templates, "views/includes/*.html", "views/*.html")
	app.SetHTMLTemplate(template)
	// 缓存中间件
	app.Use(middlewares.CacheMiddleware())
	// 静态文件路由
	app.StaticFS("/public", mustFS())
	app.GET("/favicon.ico", func(c *gin.Context) {
		file, _ := staticFS.ReadFile("views/public/icon/favicon.ico")
		c.Data(http.StatusOK, "image/x-icon", file)
	})
	app.SetTrustedProxies([]string{"127.0.0.1"})
	routes.RegisterRoutes(app)
	go services.RenewSSL()
	server := endless.NewServer("0.0.0.0:7777", app)

	server.BeforeBegin = func(add string) {
		log.Printf("[%d]: 服务器启动, [PPID]: %d", syscall.Getpid(), syscall.Getppid())
	}

	server.SignalHooks[endless.PRE_SIGNAL][syscall.SIGHUP] = append(
		server.SignalHooks[endless.PRE_SIGNAL][syscall.SIGHUP],
		func() {
			log.Printf("[%d]: 发送重启信号, 重启 ing...", syscall.Getpid())
		})

	server.SignalHooks[endless.POST_SIGNAL][syscall.SIGHUP] = append(
		server.SignalHooks[endless.POST_SIGNAL][syscall.SIGHUP],
		func() {
			log.Printf("[+] 重启更新完毕")
		})

	err := server.ListenAndServe()
	if err != nil {
		log.Println(err)
	}
	log.Printf("[%d]: 服务器关闭", syscall.Getpid())
	os.Exit(0)
}
