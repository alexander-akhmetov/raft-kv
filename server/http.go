package server

import (
	"fmt"
	"log"

	"raft-kv/node"

	"github.com/gin-gonic/gin"
)

type joinData struct {
	Address string `json:"address"`
}

// joinView handles 'join' request from another nodes
// if some node wants to join to the cluster it must be added by leader
// so this node sends a POST request to the leader with it's address and the leades adds it as a voter
// if this node os not a leader, it forwards request to current cluster leader
func joinView(storage *node.RStorage) func(*gin.Context) {
	view := func(c *gin.Context) {
		var data joinData
		err := c.BindJSON(&data)
		if err != nil {
			log.Printf("[ERROR] Reading POST data error: %+v", err)
			c.JSON(503, gin.H{})
			return
		}

		err = storage.AddVoter(data.Address)
		if err != nil {
			c.JSON(503, gin.H{})
		} else {
			c.JSON(200, gin.H{})
		}
	}
	return view
}

func getKeyView(storage *node.RStorage) func(*gin.Context) {
	view := func(c *gin.Context) {
		key := c.Param("key")
		c.JSON(200, gin.H{
			"value": storage.Get(key),
		})
	}
	return view
}

type setKeyData struct {
	Value string `json:"value"`
}

func setKeyView(storage *node.RStorage) func(*gin.Context) {
	view := func(c *gin.Context) {
		key := c.Param("key")
		var data setKeyData
		err := c.BindJSON(&data)
		if err != nil {
			log.Printf("[ERROR] Reading POST data error: %+v", err)
			c.JSON(503, gin.H{})
		}

		err = storage.Set(key, data.Value)
		if err != nil {
			c.JSON(503, gin.H{
				"code":  "some_code", // todo :)
				"error": fmt.Sprintf("%+v", err),
			})
		} else {
			c.JSON(200, gin.H{
				"value": data.Value,
			})
		}
	}
	return view
}

func setupRouter(raftNode *node.RStorage) *gin.Engine {
	router := gin.Default()

	router.POST("/cluster/join/", joinView(raftNode))
	router.GET("/keys/:key/", getKeyView(raftNode))
	router.POST("/keys/:key/", setKeyView(raftNode))

	return router
}

// RunHTTPServer starts HTTP server
func RunHTTPServer(raftNode *node.RStorage) {
	router := setupRouter(raftNode)
	router.Run() // listen and serve on 0.0.0.0:8080
}
