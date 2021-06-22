package retry

import (
	"fmt"
	"time"

	"github.com/bytedance/gopkg/lang/fastrand"
)

const maxFailureRetryTimes = 5

// NewFailurePolicy init failure retry policy
func NewFailurePolicy() *FailurePolicy {
	p := &FailurePolicy{
		StopPolicy: StopPolicy{
			MaxRetryTimes:    2,
			DisableChainStop: false,
			CBPolicy: CBPolicy{
				ErrorRate: defaultCBErrRate,
				MinSample: defaultCBMinSample,
			},
		},
		BackOffPolicy: &BackOffPolicy{BackOffType: NoneBackOffType},
	}
	return p
}

// WithMaxRetryTimes default is 2, not include first call
func (p *FailurePolicy) WithMaxRetryTimes(retryTimes int) {
	if retryTimes < 0 || retryTimes > maxFailureRetryTimes {
		panic(fmt.Errorf("maxFailureRetryTimes is %d", maxFailureRetryTimes))
	}
	p.StopPolicy.MaxRetryTimes = retryTimes
}

// WithMaxDurationMS include first call and retry call, if the total duration exceed limit then stop retry
func (p *FailurePolicy) WithMaxDurationMS(maxMS uint32) {
	p.StopPolicy.MaxDurationMS = maxMS
}

// DisableChainRetryStop disables chain stop
func (p *FailurePolicy) DisableChainRetryStop() {
	p.StopPolicy.DisableChainStop = true
}

// WithDDLStop sets ddl stop to true.
func (p *FailurePolicy) WithDDLStop() {
	p.StopPolicy.DDLStop = true
}

// WithFixedBackOff set fixed time.
func (p *FailurePolicy) WithFixedBackOff(fixMS int) {
	if err := checkFixedBackOff(fixMS); err != nil {
		panic(err)
	}
	p.BackOffPolicy = &BackOffPolicy{FixedBackOffType, make(map[BackOffCfgKey]float64, 1)}
	p.BackOffPolicy.CfgItems[FixMSBackOffCfgKey] = float64(fixMS)
}

// WithRandomBackOff sets the random time.
func (p *FailurePolicy) WithRandomBackOff(minMS int, maxMS int) {
	if err := checkRandomBackOff(minMS, maxMS); err != nil {
		panic(err)
	}
	p.BackOffPolicy = &BackOffPolicy{RandomBackOffType, make(map[BackOffCfgKey]float64, 2)}
	p.BackOffPolicy.CfgItems[MinMSBackOffCfgKey] = float64(minMS)
	p.BackOffPolicy.CfgItems[MaxMSBackOffCfgKey] = float64(maxMS)
}

// WithRetryBreaker sets the error rate.
func (p *FailurePolicy) WithRetryBreaker(errRate float64) {
	p.StopPolicy.CBPolicy.ErrorRate = errRate
	if err := checkCBErrorRate(&p.StopPolicy.CBPolicy); err != nil {
		panic(err)
	}
}

// WithRetrySameNode sets to retry the same node.
func (p *FailurePolicy) WithRetrySameNode() {
	p.RetrySameNode = true
}

// String prints human readable information.
func (p FailurePolicy) String() string {
	return fmt.Sprintf("{StopPolicy:%+v BackOffPolicy:%+v RetrySameNode:%+v}", p.StopPolicy, p.BackOffPolicy, p.RetrySameNode)
}

// BackOff is the interface of back off implements
type BackOff interface {
	Wait(callTimes int)
}

// NoneBackOff means there's no backoff.
var NoneBackOff = &noneBackOff{}

func newFixedBackOff(fixMS int) BackOff {
	return &fixedBackOff{
		fixMS: fixMS,
	}
}

func newRandomBackOff(minMS int, maxMS int) BackOff {
	return &randomBackOff{
		minMS: minMS,
		maxMS: maxMS,
	}
}

type noneBackOff struct {
}

// String prints human readable information.
func (p noneBackOff) String() string {
	return "NoneBackOff"
}

// Wait implements the BackOff interface.
func (p *noneBackOff) Wait(callTimes int) {
}

type fixedBackOff struct {
	fixMS int
}

// Wait implements the BackOff interface.
func (p *fixedBackOff) Wait(callTimes int) {
	time.Sleep(time.Duration(p.fixMS) * time.Millisecond)
}

// String prints human readable information.
func (p fixedBackOff) String() string {
	return fmt.Sprintf("FixedBackOff(%dms)", p.fixMS)
}

type randomBackOff struct {
	minMS int
	maxMS int
}

func (p *randomBackOff) randomDuration() time.Duration {
	return time.Duration(p.minMS+fastrand.Intn(p.maxMS-p.minMS)) * time.Millisecond
}

// Wait implements the BackOff interface.
func (p *randomBackOff) Wait(callTimes int) {
	time.Sleep(p.randomDuration())
}

// String prints human readable information.
func (p randomBackOff) String() string {
	return fmt.Sprintf("RandomBackOff(%dms-%dms)", p.minMS, p.maxMS)
}

func checkFixedBackOff(fixMS int) error {
	if fixMS == 0 {
		return fmt.Errorf("invalid FixedBackOff, fixMS=%d", fixMS)
	}
	return nil
}

func checkRandomBackOff(minMS int, maxMS int) error {
	if maxMS <= minMS {
		return fmt.Errorf("invalid RandomBackOff, minMS=%d, maxMS=%d", minMS, maxMS)
	}
	return nil
}
