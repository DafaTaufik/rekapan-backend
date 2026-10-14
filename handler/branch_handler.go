package handler

import (
	"math"
	"net/http"
	"rekap-backend/config"
	"time"

	"github.com/gin-gonic/gin"
)

// BranchResult holds aggregated statistics per branch
type BranchResult struct {
	BranchID          int     `json:"branch_id"`
	TotalTransactions int64   `json:"total_transactions"`
	TotalRevenue      float64 `json:"total_revenue"`
	TotalKg           float64 `json:"total_kg"`
}

// BranchesResponse is the full response shape for GET /api/branches
type BranchesResponse struct {
	Period            PeriodInfo     `json:"period"`
	TotalRevenue      float64        `json:"total_revenue"`
	TotalTransactions int64          `json:"total_transactions"`
	TotalKg           float64        `json:"total_kg"`
	Data              []BranchResult `json:"data"`
}

// PeriodInfo holds the date range used in the query
type PeriodInfo struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// GetBranches returns a list of branches with their transaction statistics
// for a given date range. Defaults to yesterday if no date params provided.
//
// Query params:
//   - start_date: YYYY-MM-DD (optional, default: yesterday)
//   - end_date:   YYYY-MM-DD (optional, default: yesterday)
//
// Validation:
//   - Format must be YYYY-MM-DD
//   - start_date must be <= end_date
//   - Range must not exceed 365 days
func GetBranches(c *gin.Context) {
	const layout = "2006-01-02"
	const maxRangeDays = 365

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	defaultDate := yesterday.Format(layout)

	startStr := c.DefaultQuery("start_date", defaultDate)
	endStr := c.DefaultQuery("end_date", defaultDate)

	// Parse start_date
	startDate, err := time.Parse(layout, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format, use: YYYY-MM-DD"})
		return
	}

	// Parse end_date
	endDate, err := time.Parse(layout, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format, use: YYYY-MM-DD"})
		return
	}

	// Validate: start_date must be <= end_date
	if startDate.After(endDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date cannot be after end_date"})
		return
	}

	// Validate: max range 365 days
	if int(endDate.Sub(startDate).Hours()/24) >= maxRangeDays {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Date range cannot exceed 365 days"})
		return
	}

	// end_date is inclusive — query up to end of that day
	endDateExclusive := endDate.AddDate(0, 0, 1)

	// Query per-branch stats
	var branches []BranchResult
	result := config.DB.Table("transactions").
		Select(`
			branch_id,
			COUNT(*) as total_transactions,
			COALESCE(SUM(total), 0) as total_revenue,
			COALESCE(SUM(jumlah_kg), 0) as total_kg
		`).
		Where("tanggal_masuk >= ? AND tanggal_masuk < ?", startDate, endDateExclusive).
		Group("branch_id").
		Order("branch_id ASC").
		Scan(&branches)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch branches"})
		return
	}

	// Calculate grand totals across all branches
	var totalRevenue float64
	var totalTransactions int64
	var totalKg float64
	for i, b := range branches {
		totalRevenue += b.TotalRevenue
		totalTransactions += b.TotalTransactions
		totalKg += b.TotalKg
		// Round per-branch kg to avoid float64 drift in individual rows
		branches[i].TotalKg = math.Round(b.TotalKg*100) / 100
	}

	// Round grand total as well (sum of already-rounded values can still drift)
	totalKg = math.Round(totalKg*100) / 100

	c.JSON(http.StatusOK, BranchesResponse{
		Period: PeriodInfo{
			StartDate: startStr,
			EndDate:   endStr,
		},
		TotalRevenue:      totalRevenue,
		TotalTransactions: totalTransactions,
		TotalKg:           totalKg,
		Data:              branches,
	})
}
