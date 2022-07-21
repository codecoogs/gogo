package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func getPing(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "pong",
    })
}

func main() {
    router := gin.Default()
    
    proxies := []string{"127.0.0.1"}
    router.SetTrustedProxies(proxies)

    router.GET("/ping", getPing)

    host := "localhost"
    port := "8080"
    url := host + ":" + port

    router.Run(url)
}
