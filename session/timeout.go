package session

import "time"

// processingTime is to accomodate for computational and communications delays.
// This also includes the timespent spent waiting for a mutex.
var processingTime = 5 * time.Second

type timeoutConfig struct {
	onChainTx time.Duration
	response  time.Duration
}

func (t timeoutConfig) proposeCh(challegeDurSecs uint64) time.Duration {
	challegeDur := time.Duration(challegeDurSecs) * time.Second
	return 3*t.response + 2*t.onChainTx + 1*challegeDur + processingTime
}

func (t timeoutConfig) respChProposalAccept(challegeDurSecs uint64) time.Duration {
	return t.proposeCh(challegeDurSecs)
}

func (t timeoutConfig) respChProposalReject() time.Duration {
	return t.response + processingTime
}
func (t timeoutConfig) chUpdate() time.Duration {
	return t.response + processingTime
}

func (t timeoutConfig) respChUpdateAccept() time.Duration {
	return t.response + processingTime
}

func (t timeoutConfig) respChUpdateReject() time.Duration {
	return t.response + processingTime
}

func (t timeoutConfig) closeCh(challegeDurSecs uint64) time.Duration {
	challegeDur := time.Duration(challegeDurSecs) * time.Second
	return 1*t.response + 3*t.onChainTx + 1*challegeDur + processingTime
}
