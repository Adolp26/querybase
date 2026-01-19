package handlers

import (
	"fmt"
	"net/http"

	"github.com/adolp26/querybase/internal/services"
	"github.com/gin-gonic/gin"
)

type EmployeeHandler struct {
	queryService *services.QueryService
}

func NewEmployeeHandler(queryService *services.QueryService) *EmployeeHandler {
	return &EmployeeHandler{
		queryService: queryService,
	}
}

func (h *EmployeeHandler) GetAll(c *gin.Context) {
	ctx := c.Request.Context()
	cacheKey := "employees:all"

	query := `SELECT employee_id, first_name, last_name, email, department_id
			  FROM employees
			  WHERE ROWNUM <= 100`

	results, err := h.queryService.ExecuteQuery(ctx, cacheKey, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"count": len(results),
	})
}

func (h *EmployeeHandler) GetByDepartment(c *gin.Context) {
	ctx := c.Request.Context()
	department := c.Param("department")

	cacheKey := fmt.Sprintf("employees:department:%s", department)

	query := `SELECT employee_id, first_name, last_name, email, department_id
			  FROM employees
			  WHERE department_id = :1`

	results, err := h.queryService.ExecuteQuery(ctx, cacheKey, query, department)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       results,
		"count":      len(results),
		"department": department,
	})
}
