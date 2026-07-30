package progress

type Phase string

const (
	PhaseDiscoverModules Phase = "Discovering modules"
	PhaseCoverageTests   Phase = "Running coverage tests"
	PhaseComplexity      Phase = "Analyzing complexity"
	PhaseProcessing      Phase = "Processing results"
)

type Reporter interface {
	StartPhase(name Phase, total int)
	Advance(n int)
	SetDetail(detail string)
	SetTotal(total int)
	FinishPhase()
	Done()
	Errored()
}

type NoopReporter struct{}

func (NoopReporter) StartPhase(Phase, int) {}
func (NoopReporter) Advance(int)           {}
func (NoopReporter) SetDetail(string)      {}
func (NoopReporter) SetTotal(int)          {}
func (NoopReporter) FinishPhase()          {}
func (NoopReporter) Done()                 {}
func (NoopReporter) Errored()              {}
