package main

import (
	"fmt"
	"log"

	"github.com/adolp26/querybase/internal/database"
	"github.com/adolp26/querybase/internal/handlers"
	"github.com/adolp26/querybase/internal/repository"
	"github.com/adolp26/querybase/internal/services"
	"github.com/adolp26/querybase/pkg/config"
	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("📖 Carregando configurações...")
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("❌ Erro ao carregar config: %v", err)
	}
	fmt.Println("✅ Configurações carregadas")

	fmt.Println("🔌 Conectando no Redis...")
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("❌ Erro ao conectar no Redis: %v", err)
	}
	defer redisClient.Close()
	fmt.Println("✅ Redis conectado")

	fmt.Println("🔌 Conectando no Oracle...")
	oracleClient, err := database.NewOracleDataSource(cfg.Oracle)
	if err != nil {
		log.Fatalf("❌ Erro ao conectar no Oracle: %v", err)
	}
	defer oracleClient.Close()
	fmt.Println("✅ Oracle conectado")

	fmt.Println("🔌 Conectando no PostgreSQL...")
	postgresClient, err := database.NewPostgresClient(cfg.Postgres)
	if err != nil {
		log.Fatalf("❌ Erro ao conectar no PostgreSQL: %v", err)
	}
	defer postgresClient.Close()
	fmt.Println("✅ PostgreSQL conectado")

	cacheService := services.NewCacheService(redisClient)
	queryService := services.NewQueryService(oracleClient, cacheService)

	queryRepo := repository.NewQueryRepository(postgresClient.GetDB())

	testHandler := handlers.NewTestHandler(cacheService)
	employeeHandler := handlers.NewEmployeeHandler(queryService)

	dynamicHandler := handlers.NewDynamicQueryHandler(queryRepo, queryService, cacheService)

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Health check
	router.GET("/health", handlers.HealthCheck)

	// Endpoints de teste/legado
	router.GET("/api/test", testHandler.GetTestData)
	router.GET("/api/employees", employeeHandler.GetAll)
	router.GET("/api/employees/department/:department", employeeHandler.GetByDepartment)

	router.GET("/api/queries", dynamicHandler.ListQueries)
	router.GET("/api/query/:slug", dynamicHandler.Execute)

	// Iniciar servidor
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	fmt.Println("")
	fmt.Printf("🚀 QueryBase API rodando em http://localhost%s\n", addr)
	fmt.Println("")
	fmt.Println("📚 Endpoints disponíveis:")
	fmt.Println("   GET /health                              - Health check")
	fmt.Println("   GET /api/test                            - Teste de cache")
	fmt.Println("   GET /api/employees                       - Listar funcionários")
	fmt.Println("   GET /api/employees/department/:dept      - Funcionários por depto")
	fmt.Println("")
	fmt.Println("   GET /api/queries                         - Listar queries disponíveis")
	fmt.Println("   GET /api/query/:slug                     - Executar query dinâmica")
	fmt.Println("")
	fmt.Println("📝 Exemplos:")
	fmt.Println("   curl http://localhost:8080/api/queries")
	fmt.Println("   curl http://localhost:8080/api/query/employees-all")
	fmt.Println("   curl \"http://localhost:8080/api/query/employees-by-department?department=10\"")
	fmt.Println("")

	if err := router.Run(addr); err != nil {
		log.Fatalf("❌ Erro ao iniciar servidor: %v", err)
	}
}
