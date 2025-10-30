package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	zbc "github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/gin-gonic/gin"
	"github.com/pflow/shared/config"
	"github.com/pflow/shared/httpx"
)

type deployRequest struct {
	BPMN string `json:"bpmn" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type startRequest struct {
	ProcessKey string                 `json:"processKey" binding:"required"`
	Variables  map[string]interface{} `json:"variables"`
}

func main() {
	cfg := config.Load()

	server := httpx.New()

	api := server.Engine.Group("/workflow")
	{
		api.POST("/deploy", deployWorkflow(cfg.CamundaURL))
		api.POST("/start", startWorkflow(cfg.CamundaURL))
	}

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("workflow service listening on %s", addr)
	if err := server.Start(addr); err != nil {
		log.Fatalf("workflow service stopped: %v", err)
	}
}

func newClient(camundaURL string) (zbc.Client, error) {
	return zbc.NewClient(&zbc.ClientConfig{GatewayAddress: camundaURL})
}

func deployWorkflow(camundaURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload deployRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		client, err := newClient(camundaURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		_, err = client.NewDeployResourceCommand().AddResource([]byte(payload.BPMN), payload.Name).Send(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"status": "deployed", "name": payload.Name})
	}
}

func startWorkflow(camundaURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload startRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		client, err := newClient(camundaURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		resp, err := client.NewCreateInstanceCommand().BPMNProcessId(payload.ProcessKey).LatestVersion().VariablesFromMap(payload.Variables).Send(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"instanceKey": resp.GetProcessInstanceKey()})
	}
}
