package metrics

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type FilterMetrics struct {
	mu                    sync.RWMutex
	totalOperations       int64
	totalInputBytes       int64
	totalOutputBytes      int64
	totalFilterTime       time.Duration
	operationMetrics      map[string]*OperationMetric
	filterEffectiveness   map[string]*FilterEffectiveness
	contextReductionStats *ContextReductionStats
}

type OperationMetric struct {
	Name              string        `json:"name"`
	Count             int64         `json:"count"`
	TotalInputBytes   int64         `json:"totalInputBytes"`
	TotalOutputBytes  int64         `json:"totalOutputBytes"`
	TotalFilterTime   time.Duration `json:"totalFilterTime"`
	AverageInputSize  int64         `json:"averageInputSize"`
	AverageOutputSize int64         `json:"averageOutputSize"`
	AverageFilterTime time.Duration `json:"averageFilterTime"`
	ReductionRatio    float64       `json:"reductionRatio"`
	LastUsed          time.Time     `json:"lastUsed"`
}

type FilterEffectiveness struct {
	FilterName         string    `json:"filterName"`
	TotalApplications  int64     `json:"totalApplications"`
	LinesFiltered      int64     `json:"linesFiltered"`
	LinesPreserved     int64     `json:"linesPreserved"`
	BytesRemoved       int64     `json:"bytesRemoved"`
	AverageReduction   float64   `json:"averageReduction"`
	LastApplication    time.Time `json:"lastApplication"`
}

type ContextReductionStats struct {
	OverallReduction     float64           `json:"overallReduction"`
	TargetReduction      float64           `json:"targetReduction"`
	ReductionByCommand   map[string]float64 `json:"reductionByCommand"`
	TokensSaved          int64             `json:"tokensSaved"`
	EstimatedCostSavings float64           `json:"estimatedCostSavings"`
	LastCalculated       time.Time         `json:"lastCalculated"`
}

type FilteringResult struct {
	Operation       string
	InputSize       int64
	OutputSize      int64
	FilterTime      time.Duration
	FiltersApplied  []string
	LinesFiltered   int64
	LinesPreserved  int64
	ReductionRatio  float64
}

func NewFilterMetrics() *FilterMetrics {
	return &FilterMetrics{
		operationMetrics:      make(map[string]*OperationMetric),
		filterEffectiveness:   make(map[string]*FilterEffectiveness),
		contextReductionStats: &ContextReductionStats{
			TargetReduction:    0.9, // 90% target reduction
			ReductionByCommand: make(map[string]float64),
		},
	}
}

func (fm *FilterMetrics) RecordFilteringResult(result *FilteringResult) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	now := time.Now()
	
	// Update overall stats
	fm.totalOperations++
	fm.totalInputBytes += result.InputSize
	fm.totalOutputBytes += result.OutputSize
	fm.totalFilterTime += result.FilterTime
	
	// Update operation-specific metrics
	opMetric, exists := fm.operationMetrics[result.Operation]
	if !exists {
		opMetric = &OperationMetric{Name: result.Operation}
		fm.operationMetrics[result.Operation] = opMetric
	}
	
	opMetric.Count++
	opMetric.TotalInputBytes += result.InputSize
	opMetric.TotalOutputBytes += result.OutputSize
	opMetric.TotalFilterTime += result.FilterTime
	opMetric.LastUsed = now
	
	// Calculate averages
	if opMetric.Count > 0 {
		opMetric.AverageInputSize = opMetric.TotalInputBytes / opMetric.Count
		opMetric.AverageOutputSize = opMetric.TotalOutputBytes / opMetric.Count
		opMetric.AverageFilterTime = opMetric.TotalFilterTime / time.Duration(opMetric.Count)
		
		if opMetric.TotalInputBytes > 0 {
			opMetric.ReductionRatio = 1.0 - (float64(opMetric.TotalOutputBytes) / float64(opMetric.TotalInputBytes))
		}
	}
	
	// Update filter effectiveness
	for _, filterName := range result.FiltersApplied {
		filterEff, exists := fm.filterEffectiveness[filterName]
		if !exists {
			filterEff = &FilterEffectiveness{FilterName: filterName}
			fm.filterEffectiveness[filterName] = filterEff
		}
		
		filterEff.TotalApplications++
		filterEff.LinesFiltered += result.LinesFiltered
		filterEff.LinesPreserved += result.LinesPreserved
		filterEff.BytesRemoved += result.InputSize - result.OutputSize
		filterEff.LastApplication = now
		
		// Calculate average reduction
		if filterEff.TotalApplications > 0 {
			totalLines := filterEff.LinesFiltered + filterEff.LinesPreserved
			if totalLines > 0 {
				filterEff.AverageReduction = float64(filterEff.LinesFiltered) / float64(totalLines)
			}
		}
	}
	
	// Update context reduction stats
	fm.updateContextReductionStats(result)
}

func (fm *FilterMetrics) updateContextReductionStats(result *FilteringResult) {
	stats := fm.contextReductionStats
	
	// Update overall reduction
	if fm.totalInputBytes > 0 {
		stats.OverallReduction = 1.0 - (float64(fm.totalOutputBytes) / float64(fm.totalInputBytes))
	}
	
	// Update command-specific reduction
	stats.ReductionByCommand[result.Operation] = result.ReductionRatio
	
	// Estimate tokens saved (rough estimate: 4 chars per token)
	stats.TokensSaved += (result.InputSize - result.OutputSize) / 4
	
	// Estimate cost savings (rough estimate: $0.000003 per token for Claude)
	stats.EstimatedCostSavings = float64(stats.TokensSaved) * 0.000003
	
	stats.LastCalculated = time.Now()
}

func (fm *FilterMetrics) GetOverallStats() OverallStats {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	var overallReduction float64
	var averageFilterTime time.Duration
	
	if fm.totalInputBytes > 0 {
		overallReduction = 1.0 - (float64(fm.totalOutputBytes) / float64(fm.totalInputBytes))
	}
	
	if fm.totalOperations > 0 {
		averageFilterTime = fm.totalFilterTime / time.Duration(fm.totalOperations)
	}
	
	return OverallStats{
		TotalOperations:    fm.totalOperations,
		TotalInputBytes:    fm.totalInputBytes,
		TotalOutputBytes:   fm.totalOutputBytes,
		OverallReduction:   overallReduction,
		AverageFilterTime:  averageFilterTime,
		TargetReduction:    fm.contextReductionStats.TargetReduction,
		ReductionAchieved:  overallReduction >= fm.contextReductionStats.TargetReduction,
		TokensSaved:        fm.contextReductionStats.TokensSaved,
		EstimatedSavings:   fm.contextReductionStats.EstimatedCostSavings,
	}
}

type OverallStats struct {
	TotalOperations   int64         `json:"totalOperations"`
	TotalInputBytes   int64         `json:"totalInputBytes"`
	TotalOutputBytes  int64         `json:"totalOutputBytes"`
	OverallReduction  float64       `json:"overallReduction"`
	AverageFilterTime time.Duration `json:"averageFilterTime"`
	TargetReduction   float64       `json:"targetReduction"`
	ReductionAchieved bool          `json:"reductionAchieved"`
	TokensSaved       int64         `json:"tokensSaved"`
	EstimatedSavings  float64       `json:"estimatedSavings"`
}

func (fm *FilterMetrics) GetOperationStats() []*OperationMetric {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	stats := make([]*OperationMetric, 0, len(fm.operationMetrics))
	for _, metric := range fm.operationMetrics {
		// Create a copy to avoid race conditions
		metricCopy := *metric
		stats = append(stats, &metricCopy)
	}
	
	return stats
}

func (fm *FilterMetrics) GetFilterEffectiveness() []*FilterEffectiveness {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	effectiveness := make([]*FilterEffectiveness, 0, len(fm.filterEffectiveness))
	for _, eff := range fm.filterEffectiveness {
		// Create a copy to avoid race conditions
		effCopy := *eff
		effectiveness = append(effectiveness, &effCopy)
	}
	
	return effectiveness
}

func (fm *FilterMetrics) GetContextReductionStats() *ContextReductionStats {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	// Create a deep copy
	statsCopy := &ContextReductionStats{
		OverallReduction:     fm.contextReductionStats.OverallReduction,
		TargetReduction:      fm.contextReductionStats.TargetReduction,
		ReductionByCommand:   make(map[string]float64),
		TokensSaved:          fm.contextReductionStats.TokensSaved,
		EstimatedCostSavings: fm.contextReductionStats.EstimatedCostSavings,
		LastCalculated:       fm.contextReductionStats.LastCalculated,
	}
	
	for cmd, reduction := range fm.contextReductionStats.ReductionByCommand {
		statsCopy.ReductionByCommand[cmd] = reduction
	}
	
	return statsCopy
}

func (fm *FilterMetrics) GetDetailedReport() DetailedReport {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	return DetailedReport{
		OverallStats:           fm.GetOverallStats(),
		OperationMetrics:       fm.GetOperationStats(),
		FilterEffectiveness:    fm.GetFilterEffectiveness(),
		ContextReductionStats:  fm.GetContextReductionStats(),
		ReportGeneratedAt:      time.Now(),
	}
}

type DetailedReport struct {
	OverallStats          OverallStats             `json:"overallStats"`
	OperationMetrics      []*OperationMetric       `json:"operationMetrics"`
	FilterEffectiveness   []*FilterEffectiveness   `json:"filterEffectiveness"`
	ContextReductionStats *ContextReductionStats   `json:"contextReductionStats"`
	ReportGeneratedAt     time.Time                `json:"reportGeneratedAt"`
}

func (fm *FilterMetrics) ExportToJSON() ([]byte, error) {
	report := fm.GetDetailedReport()
	return json.MarshalIndent(report, "", "  ")
}

func (fm *FilterMetrics) Reset() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	fm.totalOperations = 0
	fm.totalInputBytes = 0
	fm.totalOutputBytes = 0
	fm.totalFilterTime = 0
	fm.operationMetrics = make(map[string]*OperationMetric)
	fm.filterEffectiveness = make(map[string]*FilterEffectiveness)
	fm.contextReductionStats = &ContextReductionStats{
		TargetReduction:    0.9,
		ReductionByCommand: make(map[string]float64),
	}
}

func (fm *FilterMetrics) GetSummaryString() string {
	stats := fm.GetOverallStats()
	
	status := "❌ Below Target"
	if stats.ReductionAchieved {
		status = "✅ Target Achieved"
	}
	
	return fmt.Sprintf(
		"Filter Performance Summary:\n"+
			"  Operations: %d\n"+
			"  Overall Reduction: %.1f%% (Target: %.1f%%) %s\n"+
			"  Tokens Saved: %d\n"+
			"  Estimated Savings: $%.4f\n"+
			"  Average Filter Time: %v\n",
		stats.TotalOperations,
		stats.OverallReduction*100,
		stats.TargetReduction*100,
		status,
		stats.TokensSaved,
		stats.EstimatedSavings,
		stats.AverageFilterTime,
	)
}