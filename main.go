package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"main.go/controllers"
	"main.go/infra"
	"main.go/models"
	"main.go/repositories"
	"main.go/services"
)

func main() {
	infra.Initialize()
	db := infra.SetupDB()
	items := []*models.Client{}
	userRepository := repositories.NewUserRepository(db)
	userService := services.NewUserService(userRepository)
	userMemoryRepository := repositories.NewUserMemoryRepository(items)
	userMemoryService := services.NewUserMemoryService(userMemoryRepository)
	userController := controllers.NewUserController(userService, userMemoryService)
	// err := userRepository.DeleteAll()
	// if err != nil {
	// 	fmt.Println("DeleteAll miss")
	// }
	r := gin.Default()
	r.SetTrustedProxies([]string{"192.168.1.012"})

	r.Use(cors.New(cors.Config{
		// AllowOrigins: []string{"*"},
		AllowOrigins:     []string{"http://www.touhobby.com:8083", "http://192.168.0.12:8083"},
		AllowMethods:     []string{"GET", "POST", "PUT"},            // 単純メソッドのみ許可
		AllowHeaders:     []string{"Content-Type", "Authorization"}, // 必要なヘッダーのみ許可
		AllowCredentials: true,
		MaxAge:           86400, // プリフライトリクエストを24時間キャッシュ
	}))

	// userRouter := r.Group("/airhockey")
	// userRouter.GET("/create", userController.CreateRoomNum)
	// userRouter.POST("/enter", userController.EnterRoom)
	r.GET("/ws", userController.HandleWebSocket)
	r.Run(":8082") // localhost:8082 でサーバーを立てます。
}
