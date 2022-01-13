package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/qfdk/nginx-proxy-manager/app/controllers"
)

func sslRouter(engine *gin.Engine) {
	engine.GET("/ssl", controllers.SSLDirs)
	engine.GET("/ssl/renew", controllers.IssueCert)
	engine.GET("/ssl/info", controllers.CertInfo)
	engine.GET("/ssl/delete", controllers.DeleteSSL)
}
