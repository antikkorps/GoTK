package bench

import (
	"sort"
	"time"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/proxy"
)

// LatencyReport holds timing statistics from repeated filter chain executions.
type LatencyReport struct {
	Iterations int
	InputBytes int
	Min        time.Duration
	Max        time.Duration
	Mean       time.Duration
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
}

// MeasureLatency runs the filter chain N times and reports timing stats.
func MeasureLatency(cfg *config.Config, input string, cmdType detect.CmdType, iterations int) LatencyReport {
	if iterations <= 0 {
		iterations = 1
	}

	chain := proxy.BuildChain(cfg, cmdType, cfg.General.MaxLines)

	durations := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		_ = chain.Apply(input)
		durations[i] = time.Since(start)
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return LatencyReport{
		Iterations: iterations,
		InputBytes: len(input),
		Min:        durations[0],
		Max:        durations[len(durations)-1],
		Mean:       total / time.Duration(iterations),
		P50:        percentile(durations, 50),
		P95:        percentile(durations, 95),
		P99:        percentile(durations, 99),
	}
}

// percentile returns the p-th percentile from a sorted slice of durations.
func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
