package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	arr := []string{"img1", "img2"}
	r.GET("/workloads/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"filtered_images": arr,
		})
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
