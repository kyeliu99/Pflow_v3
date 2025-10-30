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

type user struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func main() {
	cfg := config.Load()
	database.Connect()

	server := httpx.New()

	api := server.Engine.Group("/identity")
	{
		api.GET("/users", listUsers)
		api.POST("/users", createUser)
	}

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("identity service listening on %s", addr)
	if err := server.Start(addr); err != nil {
		log.Fatalf("identity service stopped: %v", err)
	}
}

func listUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []user{}})
}

func createUser(c *gin.Context) {
	var payload user
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": payload})
}
