package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/adolp26/querybase/internal/models"
	"github.com/adolp26/querybase/internal/repository"
	"github.com/adolp26/querybase/internal/services"
	"github.com/gin-gonic/gin"
)

type DynamicQueryHandler struct {
	queryRepo    *repository.QueryRepository
	queryService *services.QueryService
	cacheService *services.CacheService
}

func NewDynamicQueryHandler(
	queryRepo *repository.QueryRepository,
	queryService *services.QueryService,
	cacheService *services.CacheService,
) *DynamicQueryHandler {
	return &DynamicQueryHandler{
		queryRepo:    queryRepo,
		queryService: queryService,
		cacheService: cacheService,
	}
}

func (h *DynamicQueryHandler) Execute(c *gin.Context) {
	slug := c.Param("slug")
	startTime := time.Now()

	ctx := c.Request.Context()
	query, err := h.queryRepo.FindBySlug(ctx, slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Query não encontrada",
			"slug":    slug,
			"details": err.Error(),
		})
		return
	}

	params, validationErrors := h.extractAndValidateParams(c, query)
	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Parâmetros inválidos",
			"slug":       slug,
			"validation": validationErrors,
		})
		return
	}

	cacheKey := h.buildCacheKey(slug, params, query.Parameters)
	args := h.buildQueryArgs(params, query.Parameters)

	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(query.TimeoutSeconds)*time.Second)
	defer cancel()

	results, cacheHit, err := h.executeWithCache(queryCtx, cacheKey, query, args)
	duration := time.Since(startTime)

	go h.logExecution(query, params, duration, cacheHit, results, err, c)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "Erro ao executar query",
			"slug":     slug,
			"details":  err.Error(),
			"duration": duration.String(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": results,
		"meta": gin.H{
			"slug":       slug,
			"name":       query.Name,
			"count":      len(results),
			"cache_hit":  cacheHit,
			"duration":   duration.String(),
			"parameters": params,
		},
	})
}

func (h *DynamicQueryHandler) extractAndValidateParams(
	c *gin.Context,
	query *models.Query,
) (map[string]interface{}, map[string]string) {
	params := make(map[string]interface{})
	errors := make(map[string]string)

	for _, p := range query.Parameters {
		rawValue := c.Query(p.Name)

		if rawValue == "" {
			if p.IsRequired {
				errors[p.Name] = "parâmetro obrigatório não fornecido"
				continue
			}
			if p.DefaultValue != nil {
				rawValue = *p.DefaultValue
			} else {
				continue
			}
		}

		converted, err := h.convertParamType(rawValue, p.ParamType)
		if err != nil {
			errors[p.Name] = fmt.Sprintf("tipo inválido: esperado %s, erro: %s", p.ParamType, err.Error())
			continue
		}

		params[p.Name] = converted
	}

	return params, errors
}

func (h *DynamicQueryHandler) convertParamType(value string, paramType string) (interface{}, error) {
	switch paramType {
	case "string":
		return value, nil
	case "integer":
		return strconv.Atoi(value)
	case "number":
		return strconv.ParseFloat(value, 64)
	case "boolean":
		return strconv.ParseBool(value)
	case "date":
		return time.Parse("2006-01-02", value)
	case "datetime":
		return time.Parse("2006-01-02 15:04:05", value)
	default:
		return value, nil
	}
}

func (h *DynamicQueryHandler) buildCacheKey(
	slug string,
	params map[string]interface{},
	definitions []models.QueryParameter,
) string {
	key := fmt.Sprintf("query:%s", slug)

	for _, def := range definitions {
		if val, ok := params[def.Name]; ok {
			key += fmt.Sprintf(":%s=%v", def.Name, val)
		}
	}

	return key
}

func (h *DynamicQueryHandler) buildQueryArgs(
	params map[string]interface{},
	definitions []models.QueryParameter,
) []interface{} {
	maxPos := 0
	for _, def := range definitions {
		if def.Position > maxPos {
			maxPos = def.Position
		}
	}

	args := make([]interface{}, maxPos)

	for _, def := range definitions {
		if val, ok := params[def.Name]; ok {
			args[def.Position-1] = val
		}
	}

	return args
}

func (h *DynamicQueryHandler) executeWithCache(
	ctx context.Context,
	cacheKey string,
	query *models.Query,
	args []interface{},
) ([]map[string]interface{}, bool, error) {
	var cacheHit bool
	var results []map[string]interface{}

	data, err := h.cacheService.GetOrSet(ctx, cacheKey, func() (interface{}, error) {
		cacheHit = false
		fmt.Printf("🔍 Executando query '%s' no Oracle...\n", query.Slug)
		return h.queryService.ExecuteQueryDirect(ctx, query.SQLQuery, args...)
	})

	if err != nil {
		return nil, false, err
	}

	if !cacheHit {
		cacheHit = false
	} else {
		cacheHit = true
		fmt.Printf("✅ Cache HIT para query '%s'\n", query.Slug)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, cacheHit, err
	}

	if err := json.Unmarshal(jsonData, &results); err != nil {
		return nil, cacheHit, err
	}

	return results, cacheHit, nil
}

func (h *DynamicQueryHandler) logExecution(
	query *models.Query,
	params map[string]interface{},
	duration time.Duration,
	cacheHit bool,
	results []map[string]interface{},
	execError error,
	c *gin.Context,
) {
	paramsJSON, _ := json.Marshal(params)

	var errMsg *string
	if execError != nil {
		msg := execError.Error()
		errMsg = &msg
	}

	execution := models.QueryExecution{
		QueryID:    &query.ID,
		QuerySlug:  query.Slug,
		DurationMs: int(duration.Milliseconds()),
		CacheHit:   cacheHit,
		RowCount:   len(results),
		Parameters: string(paramsJSON),
		Error:      errMsg,
		ClientIP:   stringPtr(c.ClientIP()),
		UserAgent:  stringPtr(c.Request.UserAgent()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.queryRepo.LogExecution(ctx, execution); err != nil {
		fmt.Printf("⚠️  Erro ao logar execução: %v\n", err)
	}
}

func stringPtr(s string) *string {
	return &s
}

func (h *DynamicQueryHandler) ListQueries(c *gin.Context) {
	ctx := c.Request.Context()

	queries, err := h.queryRepo.ListActive(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erro ao listar queries",
		})
		return
	}

	var endpoints []gin.H
	for _, q := range queries {
		endpoint := gin.H{
			"slug":        q.Slug,
			"name":        q.Name,
			"description": q.Description,
			"endpoint":    fmt.Sprintf("/api/query/%s", q.Slug),
			"cache_ttl":   q.CacheTTL,
			"parameters":  q.Parameters,
		}
		endpoints = append(endpoints, endpoint)
	}

	c.JSON(http.StatusOK, gin.H{
		"queries": endpoints,
		"count":   len(endpoints),
	})
}
