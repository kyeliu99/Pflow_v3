package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pflow/shared/config"
	"github.com/pflow/shared/database"
	"github.com/pflow/shared/httpx"
)

type form struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`
}

func main() {
	cfg := config.Load()
	database.Connect()

	server := httpx.New()
	api := server.Engine.Group("/forms")
	{
		api.GET("", listForms)
		api.POST("", createForm)
		api.GET(":id", getForm)
	}

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("form service listening on %s", addr)
	if err := server.Start(addr); err != nil {
		log.Fatalf("form service stopped: %v", err)
	}
}

func listForms(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []form{}})
}

func createForm(c *gin.Context) {
	var payload form
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": payload})
}

func getForm(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": form{ID: c.Param("id")}})
}
