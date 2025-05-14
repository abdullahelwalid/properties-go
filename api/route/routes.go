package route

import (
	"golang-test/api/handler"
	"golang-test/utils"

	"github.com/gin-gonic/gin"
)

func SetupRouter(userHandler *handler.UserHandler, authHandler *handler.AuthHandler, roleHandler *handler.RoleHandler, propertiesHandler *handler.PropertiesHandler, categoryHandler *handler.PropertyCategoryHandler, propertyTypesHandler *handler.PropertyTypeHandler, transactionHandler *handler.TransactionHandler) *gin.Engine {
	router := gin.Default()

	router.POST("/login", authHandler.Login)
	router.POST("/users", userHandler.CreateUser)

	authorizedRouter := router.Group("/")
	authorizedRouter.Use(utils.AuthMiddleware([]uint{uint(0)}))
	authorizedRouter.GET("/users", userHandler.GetAllUsers)
	authorizedRouter.GET("/users/:id", userHandler.GetUserById)
	authorizedRouter.GET("/properties", propertiesHandler.GetProperties)
	authorizedRouter.GET("/properties/:id", propertiesHandler.GetPropertyByID)
	authorizedRouter.GET("/categories", categoryHandler.GetCategories)
	authorizedRouter.GET("/types", propertyTypesHandler.GetTypes)
	authorizedRouter.POST("/transactions", transactionHandler.CreateTransaction)
	authorizedRouter.GET("/transactions", transactionHandler.GetUserTransactions)


	authorizedRouterAdminOrOwner := router.Group("/")
	authorizedRouterAdminOrOwner.Use(utils.AuthMiddleware([]uint{uint(2), uint(3)}))
	authorizedRouterAdminOrOwner.POST("/properties", propertiesHandler.CreateProperty)
	authorizedRouterAdminOrOwner.PUT("/properties/:id", propertiesHandler.UpdateProperty)
	authorizedRouterAdminOrOwner.DELETE("/properties/:id", propertiesHandler.DeleteProperty)



	adminAuthRoute := router.Group("/admin")
	adminAuthRoute.Use(utils.AuthMiddleware([]uint{uint(3)}))
	adminAuthRoute.POST("/categories", categoryHandler.CreateCategory)
	adminAuthRoute.POST("/types", propertyTypesHandler.CreateType)
	adminAuthRoute.POST("/roles", roleHandler.CreateRole)
	return router
}
