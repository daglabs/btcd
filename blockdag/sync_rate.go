package blockdag

import "time"

const syncRateWindowDuration = 15 * time.Minute

// addBlockProcessTimestamp adds the last block process timestamp in order to measure the recent sync rate.
//
// This function MUST be called with the DAG state lock held (for writes).
func (dag *BlockDAG) addBlockProcessTimestamp() {
	now := time.Now()
	dag.recentBlockProcessTimes = append(dag.recentBlockProcessTimes, now)
	dag.recentBlockProcessTimes = dag.recentBlockProcessTimesRelevantWindow()
}

func (dag *BlockDAG) recentBlockProcessTimesRelevantWindow() []time.Time {
	minTime := time.Now().Add(-syncRateWindowDuration)
	windowStartIndex := len(dag.recentBlockProcessTimes)
	for i, processTime := range dag.recentBlockProcessTimes {
		if processTime.After(minTime) {
			windowStartIndex = i
			break
		}
	}
	return dag.recentBlockProcessTimes[windowStartIndex:]
}

// syncRate returns the rate of processed
// blocks in the last syncRateWindowDuration
// duration.
func (dag *BlockDAG) syncRate() float64 {
	dag.RLock()
	defer dag.RUnlock()
	return float64(len(dag.recentBlockProcessTimesRelevantWindow())) / syncRateWindowDuration.Seconds()
}

// IsSyncRateBelowThreshold checks whether the sync rate
// is below an expected threshold.
func (dag *BlockDAG) IsSyncRateBelowThreshold(maxDeviation float64) bool {
	if dag.uptime() < syncRateWindowDuration {
		return false
	}

	return dag.syncRate() < 1/dag.dagParams.TargetTimePerBlock.Seconds()*maxDeviation
}

func (dag *BlockDAG) uptime() time.Duration {
	return time.Now().Sub(dag.startTime)
}
