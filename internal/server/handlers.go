package server

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"github.com/sosalejandro/testreg/internal/adapters"
	"github.com/sosalejandro/testreg/internal/app"
	"github.com/sosalejandro/testreg/internal/domain"
	"github.com/sosalejandro/testreg/internal/ports"
)

// ─── View Models ────────────────────────────────────────────────────────────

// NavItem is a single sidebar navigation link.
type NavItem struct {
	Path   string
	Icon   string
	Label  string
	Active bool
}

// StatusVM is the status bar data.
type StatusVM struct {
	TotalFeatures int
	TotalTests    int
	AtTarget      int
	AtTargetPct   int
	DomainData    []app.DomainStatusRow
}

// OverviewVM holds dashboard page data.
type OverviewVM struct {
	PriorityRings []DonutRingVM
	CoverageBars  []ProgressBarVM
	SprintTop     []SprintItemVM
}

// DonutRingVM is data for a single donut chart.
type DonutRingVM struct {
	Pct         int
	AtTarget    int
	Total       int
	DashOffset  int
	StrokeClass string
	LabelClass  string
	Label       string
}

// ProgressBarVM is data for a single coverage bar.
type ProgressBarVM struct {
	Label    string
	Pct      int
	Count    int
	PctClass string
	BarClass string
}

// SprintItemVM is a single row in the sprint priorities list.
type SprintItemVM struct {
	ID           string
	Domain       string
	Priority     string
	Score        float64
	HealthPct    int
	TargetPct    int
	PriorityDot  string
	PriorityText string
	HealthBg     string
	TargetBg     string
}

// FeaturesVM holds the features page data.
type FeaturesVM struct {
	Rows []FeatureRowVM
}

// FeatureRowVM is a single row in the features table.
type FeatureRowVM struct {
	ID           string
	Domain       string
	Priority     string
	Status       string
	UnitCovered  bool
	IntegCovered bool
	E2ECovered   bool
	PriorityDot  string
	PriorityText string
	StatusClass  string
}

// ContractVM holds the contract page data.
type ContractVM struct {
	FeatureID  string
	EntryPoint string
	LayerCount int
	Layers     []ContractLayerVM
}

// ContractLayerVM is a single layer in the contract chain.
type ContractLayerVM struct {
	Kind         string
	NodeID       string
	FunctionName string
	Signature    string
	Calls        []string
}

// DiagnoseVM holds the diagnose page data.
type DiagnoseVM struct {
	FeatureID  string
	Symptom    string
	Rule       *domain.SymptomRule
	AllRules   []*domain.SymptomRule
	CheckFiles []string
}

// DiagnoseChipsVM holds quick-pick chip groups.
type DiagnoseChipsVM struct {
	HTTP     []string
	Infra    []string
	Frontend []string
}

// SprintVM holds the sprint page data.
type SprintVM struct {
	Items []SprintItemVM
}

// PageData is the top-level template data passed to every page.
type PageData struct {
	ProjectName     string
	Version         string
	Nav             []NavItem
	Status          StatusVM
	Content         template.HTML // pre-rendered page content injected into base.html
	Overview        *OverviewVM
	Features        *FeaturesVM
	Contract        *ContractVM
	AllFeatures     []string
	DiagnoseFeature string
	DiagnoseSymptom string
	DiagnoseChips   DiagnoseChipsVM
	Diagnose        *DiagnoseVM
	Sprint          *SprintVM
	Diff            any // future
}

// ─── Base page builder ───────────────────────────────────────────────────────

func (s *Server) buildBase(active string) (*PageData, error) {
	statusResult, err := s.statusUC.Execute(s.registryDir, app.StatusFilter{})
	if err != nil {
		return nil, fmt.Errorf("loading status: %w", err)
	}

	reg, err := s.store.LoadAll(s.registryDir)
	if err != nil {
		return nil, fmt.Errorf("loading registry: %w", err)
	}

	var allIDs []string
	for _, f := range reg.AllFeatures() {
		allIDs = append(allIDs, f.ID)
	}
	sort.Strings(allIDs)

	nav := []NavItem{
		{Path: "/", Icon: "dashboard", Label: "Overview", Active: active == "overview"},
		{Path: "/features", Icon: "extension", Label: "Features", Active: active == "features"},
		{Path: "/sprint", Icon: "reorder", Label: "Sprint", Active: active == "sprint"},
		{Path: "/contract", Icon: "description", Label: "Contract", Active: active == "contract"},
		{Path: "/metrics", Icon: "analytics", Label: "Metrics", Active: active == "metrics"},
		{Path: "/diff", Icon: "difference", Label: "Diff", Active: active == "diff"},
		{Path: "/diagnose", Icon: "troubleshoot", Label: "Diagnose", Active: active == "diagnose"},
	}

	return &PageData{
		ProjectName: s.projectName,
		Version:     "1.0.4",
		Nav:         nav,
		Status:      buildStatusVM(statusResult),
		AllFeatures: allIDs,
	}, nil
}

func buildStatusVM(r *app.StatusResult) StatusVM {
	total := r.Metrics.TotalFeatures
	atTarget := r.Metrics.CoveredUnit
	pct := 0
	if total > 0 {
		pct = (atTarget * 100) / total
	}
	totalTests := r.Metrics.CoveredUnit + r.Metrics.CoveredIntegration + r.Metrics.CoveredE2E
	return StatusVM{
		TotalFeatures: total,
		TotalTests:    totalTests,
		AtTarget:      atTarget,
		AtTargetPct:   pct,
		DomainData:    r.DomainData,
	}
}

// ─── Full page handlers (first load) ────────────────────────────────────────

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := s.buildOverviewData()
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderFull(w, "overview-content", data)
}

func (s *Server) handleFeatures(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildFeaturesData(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderFull(w, "features-content", data)
}

func (s *Server) handleContract(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildContractData(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderFull(w, "contract-content", data)
}

func (s *Server) handleDiagnose(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildDiagnoseData(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderFull(w, "diagnose-content", data)
}

func (s *Server) handleSprint(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildSprintData()
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderFull(w, "sprint-content", data)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildMetricsData()
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderFull(w, "metrics-content", data)
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildBase("diff")
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderFull(w, "diff-content", data)
}

// ─── Partial handlers (htmx swaps into #page-content) ────────────────────────

func (s *Server) handleOverviewPartial(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildOverviewData()
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderPartial(w, "overview-content", data)
}

func (s *Server) handleFeaturesPartial(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildFeaturesData(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	// If htmx is targeting only the table body (filter/search).
	if r.Header.Get("HX-Target") == "feature-table-body" {
		s.renderPartial(w, "feature-rows", data.Features)
		return
	}
	s.renderPartial(w, "features-content", data)
}

func (s *Server) handleContractPartial(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildContractData(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderPartial(w, "contract-content", data)
}

func (s *Server) handleDiagnosePartial(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		diag, err := s.runDiagnose(r)
		if err != nil {
			s.serverError(w, err)
			return
		}
		s.renderPartial(w, "diagnose-result", diag)
		return
	}
	data, err := s.buildDiagnoseData(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderPartial(w, "diagnose-content", data)
}

func (s *Server) handleSprintPartial(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildSprintData()
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderPartial(w, "sprint-content", data)
}

func (s *Server) handleMetricsPartial(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildMetricsData()
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderPartial(w, "metrics-content", data)
}

func (s *Server) handleDiffPartial(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildBase("diff")
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderPartial(w, "diff-content", data)
}

// ─── API handlers ─────────────────────────────────────────────────────────────

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scanners := []ports.TestScanner{
		adapters.NewGoScanner(),
		adapters.NewVitestScanner(),
		adapters.NewPlaywrightScanner(),
		adapters.NewMaestroScanner(),
		adapters.NewJestScanner(),
		adapters.NewPythonScanner(),
	}
	scanUC := app.NewScanTestsUseCase(s.store, s.store, scanners)
	if _, err := scanUC.Execute(s.projectRoot, s.registryDir); err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<footer id="status-bar" class="fixed bottom-0 w-full h-8 z-[60] flex items-center px-4 bg-slate-900 font-mono text-[11px] text-red-400">
            <span class="material-symbols-outlined mr-2 text-sm">error</span>Scan failed: %s
        </footer>`, template.HTMLEscapeString(err.Error()))
		return
	}

	statusResult, _ := s.statusUC.Execute(s.registryDir, app.StatusFilter{})
	status := buildStatusVM(statusResult)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<footer id="status-bar" class="fixed bottom-0 w-full h-8 z-[60] flex items-center px-4 bg-slate-900 font-mono text-[11px] tabular-nums">
        <div class="flex items-center justify-between w-full">
            <div class="flex items-center gap-4">
                <div class="flex items-center gap-1.5">
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
                    <span class="text-slate-400">Scan complete</span>
                </div>
                <div class="h-3 w-[1px] bg-slate-800"></div>
                <div class="flex items-center gap-3 text-slate-300">
                    <span>%d features</span><span class="text-slate-600">•</span>
                    <span>%d tests</span><span class="text-slate-600">•</span>
                    <span class="text-emerald-400">%d%% at target</span>
                </div>
            </div>
        </div>
    </footer>`, status.TotalFeatures, status.TotalTests, status.AtTargetPct)
}

// ─── Data builders ────────────────────────────────────────────────────────────

func (s *Server) buildOverviewData() (*PageData, error) {
	data, err := s.buildBase("overview")
	if err != nil {
		return nil, err
	}

	statusResult, err := s.statusUC.Execute(s.registryDir, app.StatusFilter{})
	if err != nil {
		return nil, err
	}

	data.Overview = &OverviewVM{
		PriorityRings: buildPriorityRings(statusResult.Metrics),
		CoverageBars:  buildCoverageBars(statusResult.Metrics),
		SprintTop:     s.buildSprintItems(10),
	}
	return data, nil
}

func (s *Server) buildFeaturesData(r *http.Request) (*PageData, error) {
	data, err := s.buildBase("features")
	if err != nil {
		return nil, err
	}

	q := r.URL.Query().Get("q")
	priority := domain.Priority(r.URL.Query().Get("priority"))

	statusResult, err := s.statusUC.Execute(s.registryDir, app.StatusFilter{
		Priority: priority,
	})
	if err != nil {
		return nil, err
	}

	var rows []FeatureRowVM
	for _, f := range statusResult.Features {
		if q != "" && !strings.Contains(strings.ToLower(f.ID), strings.ToLower(q)) &&
			!strings.Contains(strings.ToLower(string(f.Priority)), strings.ToLower(q)) {
			continue
		}
		rows = append(rows, featureToVM(f))
	}

	data.Features = &FeaturesVM{Rows: rows}
	return data, nil
}

func (s *Server) buildContractData(r *http.Request) (*PageData, error) {
	data, err := s.buildBase("contract")
	if err != nil {
		return nil, err
	}

	featureID := r.URL.Query().Get("feature")
	if featureID == "" && len(data.AllFeatures) > 0 {
		featureID = data.AllFeatures[0]
	}

	if featureID != "" {
		contract, cerr := s.contractUC.Execute(s.registryDir, featureID, s.config)
		if cerr == nil && contract != nil {
			data.Contract = contractToVM(contract)
		}
	}

	return data, nil
}

func (s *Server) buildDiagnoseData(r *http.Request) (*PageData, error) {
	data, err := s.buildBase("diagnose")
	if err != nil {
		return nil, err
	}

	data.DiagnoseFeature = r.URL.Query().Get("feature")
	if data.DiagnoseFeature == "" && len(data.AllFeatures) > 0 {
		data.DiagnoseFeature = data.AllFeatures[0]
	}
	data.DiagnoseSymptom = r.URL.Query().Get("symptom")
	data.DiagnoseChips = defaultDiagnoseChips()
	return data, nil
}

func (s *Server) runDiagnose(r *http.Request) (*DiagnoseVM, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	featureID := r.FormValue("feature")
	symptom := r.FormValue("symptom")

	out, err := s.diagnoseUC.Execute(s.registryDir, featureID, symptom, s.config)
	if err != nil {
		return &DiagnoseVM{FeatureID: featureID, Symptom: symptom}, nil
	}

	return &DiagnoseVM{
		FeatureID:  out.FeatureID,
		Symptom:    out.Symptom,
		Rule:       out.Rule,
		AllRules:   out.AllRules,
		CheckFiles: out.CheckFiles,
	}, nil
}

func (s *Server) buildSprintData() (*PageData, error) {
	data, err := s.buildBase("sprint")
	if err != nil {
		return nil, err
	}
	data.Sprint = &SprintVM{Items: s.buildSprintItems(0)}
	return data, nil
}

func (s *Server) buildMetricsData() (*PageData, error) {
	data, err := s.buildBase("metrics")
	if err != nil {
		return nil, err
	}

	statusResult, err := s.statusUC.Execute(s.registryDir, app.StatusFilter{})
	if err != nil {
		return nil, err
	}

	data.Overview = &OverviewVM{
		CoverageBars: buildCoverageBars(statusResult.Metrics),
	}
	return data, nil
}

// ─── Sprint helpers ───────────────────────────────────────────────────────────

// buildSprintItems computes priority-weighted gap scores. limit=0 means all items.
func (s *Server) buildSprintItems(limit int) []SprintItemVM {
	weights := map[domain.Priority]float64{
		domain.PriorityCritical: 4,
		domain.PriorityHigh:     3,
		domain.PriorityMedium:   2,
		domain.PriorityLow:      1,
	}
	targets := map[domain.Priority]float64{
		domain.PriorityCritical: 1.0,
		domain.PriorityHigh:     0.8,
		domain.PriorityMedium:   0.6,
		domain.PriorityLow:      0.4,
	}

	type scored struct {
		f      domain.Feature
		domain string
		score  float64
	}

	reg, _ := s.store.LoadAll(s.registryDir)
	if reg == nil {
		return nil
	}

	var items []scored
	for _, d := range reg.Domains {
		for _, f := range d.Features {
			target := targets[f.Priority]
			health := featureHealth(f)
			gap := target - health
			if gap <= 0 {
				continue
			}
			items = append(items, scored{f: f, domain: d.Domain, score: weights[f.Priority] * gap})
		}
	}

	sort.Slice(items, func(i, j int) bool { return items[i].score > items[j].score })

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	out := make([]SprintItemVM, 0, len(items))
	for _, it := range items {
		target := targets[it.f.Priority]
		health := featureHealth(it.f)
		out = append(out, SprintItemVM{
			ID:           it.f.ID,
			Domain:       it.domain,
			Priority:     string(it.f.Priority),
			Score:        it.score,
			HealthPct:    int(health * 100),
			TargetPct:    int(target * 100),
			PriorityDot:  priorityDot(it.f.Priority),
			PriorityText: priorityText(it.f.Priority),
			HealthBg:     healthBarClass(health),
			TargetBg:     targetBgClass(it.f.Priority),
		})
	}
	return out
}

// ─── View model builders ──────────────────────────────────────────────────────

func buildPriorityRings(m domain.Metrics) []DonutRingVM {
	type ring struct {
		label       string
		priority    domain.Priority
		strokeClass string
		labelClass  string
	}
	rings := []ring{
		{"Critical", domain.PriorityCritical, "stroke-red-500", "text-red-500"},
		{"High", domain.PriorityHigh, "stroke-yellow-500", "text-yellow-500"},
		{"Medium", domain.PriorityMedium, "stroke-emerald-500", "text-emerald-500"},
		{"Low", domain.PriorityLow, "stroke-slate-500", "text-slate-400"},
	}

	out := make([]DonutRingVM, 0, 4)
	for _, r := range rings {
		pm := m.ByPriority[r.priority]
		pct := 0
		if pm.Total > 0 {
			pct = (pm.CoveredUnit * 100) / pm.Total
		}
		dashOffset := 88 - (88 * pct / 100)
		out = append(out, DonutRingVM{
			Pct:         pct,
			AtTarget:    pm.CoveredUnit,
			Total:       pm.Total,
			DashOffset:  dashOffset,
			StrokeClass: r.strokeClass,
			LabelClass:  r.labelClass,
			Label:       r.label,
		})
	}
	return out
}

func buildCoverageBars(m domain.Metrics) []ProgressBarVM {
	total := m.TotalFeatures
	pct := func(n int) int {
		if total == 0 {
			return 0
		}
		return (n * 100) / total
	}
	barClass := func(p int) string {
		if p >= 70 {
			return "bg-emerald-500"
		}
		if p >= 40 {
			return "bg-yellow-500"
		}
		return "bg-red-500"
	}
	pctClass := func(p int) string {
		if p >= 70 {
			return "text-emerald-500"
		}
		if p >= 40 {
			return "text-yellow-500"
		}
		return "text-red-500"
	}

	up := pct(m.CoveredUnit)
	ip := pct(m.CoveredIntegration)
	ep := pct(m.CoveredE2E)

	return []ProgressBarVM{
		{Label: "Unit Tests", Pct: up, Count: m.CoveredUnit, PctClass: pctClass(up), BarClass: barClass(up)},
		{Label: "Integration Tests", Pct: ip, Count: m.CoveredIntegration, PctClass: pctClass(ip), BarClass: barClass(ip)},
		{Label: "E2E Tests", Pct: ep, Count: m.CoveredE2E, PctClass: pctClass(ep), BarClass: barClass(ep)},
	}
}

func featureToVM(f domain.Feature) FeatureRowVM {
	unitCov := f.Coverage.Unit.Backend != nil || f.Coverage.Unit.Web != nil || f.Coverage.Unit.Mobile != nil
	integCov := f.Coverage.Integration.Backend != nil || f.Coverage.Integration.Mobile != nil
	e2eCov := f.Coverage.E2E.Web != nil || f.Coverage.E2E.Mobile != nil

	status := "missing"
	if unitCov && integCov {
		status = "covered"
	} else if unitCov || integCov || e2eCov {
		status = "partial"
	}

	return FeatureRowVM{
		ID:           f.ID,
		Priority:     string(f.Priority),
		Status:       status,
		UnitCovered:  unitCov,
		IntegCovered: integCov,
		E2ECovered:   e2eCov,
		PriorityDot:  priorityDot(f.Priority),
		PriorityText: priorityText(f.Priority),
		StatusClass:  statusBadgeClass(status),
	}
}

func contractToVM(c *domain.ContractOutput) *ContractVM {
	vm := &ContractVM{
		FeatureID:  c.FeatureID,
		EntryPoint: c.EntryPoint,
		LayerCount: len(c.Layers),
	}
	for _, l := range c.Layers {
		var calls []string
		if l.DelegateTo != "" {
			calls = append(calls, l.DelegateTo)
		}
		vm.Layers = append(vm.Layers, ContractLayerVM{
			Kind:         l.Kind,
			NodeID:       l.NodeID,
			FunctionName: l.Name,
			Signature:    l.Signature,
			Calls:        calls,
		})
	}
	return vm
}

// ─── Priority / status helpers ────────────────────────────────────────────────

func priorityDot(p domain.Priority) string {
	switch p {
	case domain.PriorityCritical:
		return "bg-red-500"
	case domain.PriorityHigh:
		return "bg-yellow-500"
	case domain.PriorityMedium:
		return "bg-emerald-500"
	default:
		return "bg-slate-500"
	}
}

func priorityText(p domain.Priority) string {
	switch p {
	case domain.PriorityCritical:
		return "text-red-500"
	case domain.PriorityHigh:
		return "text-yellow-500"
	case domain.PriorityMedium:
		return "text-emerald-500"
	default:
		return "text-slate-400"
	}
}

func statusBadgeClass(s string) string {
	switch s {
	case "covered":
		return "bg-emerald-500/10 text-emerald-400"
	case "partial":
		return "bg-yellow-500/10 text-yellow-400"
	case "failing":
		return "bg-red-500/10 text-red-400"
	default:
		return "bg-slate-800 text-slate-500"
	}
}

func healthBarClass(h float64) string {
	if h >= 0.7 {
		return "bg-emerald-500"
	}
	if h >= 0.4 {
		return "bg-yellow-500"
	}
	return "bg-red-500"
}

func targetBgClass(p domain.Priority) string {
	switch p {
	case domain.PriorityCritical:
		return "bg-red-500/30"
	case domain.PriorityHigh:
		return "bg-yellow-500/30"
	case domain.PriorityMedium:
		return "bg-emerald-500/30"
	default:
		return "bg-slate-500/30"
	}
}

// featureHealth returns a 0..1 health score approximated from coverage entries.
func featureHealth(f domain.Feature) float64 {
	score, total := 0.0, 0.0
	checkEntry := func(e *domain.CoverageEntry) {
		if e == nil {
			return
		}
		total++
		if !e.Status.IsMissing() {
			score++
		}
	}
	checkE2E := func(e *domain.E2ECoverageEntry) {
		if e == nil {
			return
		}
		total++
		if !e.Status.IsMissing() {
			score++
		}
	}
	checkEntry(f.Coverage.Unit.Backend)
	checkEntry(f.Coverage.Unit.Web)
	checkEntry(f.Coverage.Integration.Backend)
	checkE2E(f.Coverage.E2E.Web)
	if total == 0 {
		return 0
	}
	return score / total
}

func defaultDiagnoseChips() DiagnoseChipsVM {
	return DiagnoseChipsVM{
		HTTP:     []string{"401", "403", "404", "500", "409", "422", "429", "502", "503"},
		Infra:    []string{"timeout", "connection refused", "unique constraint", "deadlock", "json unmarshal", "CORS", "EOF", "TLS"},
		Frontend: []string{"TypeError", "selector not found", "hydration mismatch"},
	}
}

// ─── Render helpers ───────────────────────────────────────────────────────────

// renderFull pre-renders contentTmpl into data.Content, then renders base.html.
// This prevents multiple {{define "page-content"}} conflicts in the template set.
func (s *Server) renderFull(w http.ResponseWriter, contentTmpl string, data *PageData) {
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, contentTmpl, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	data.Content = template.HTML(buf.String()) //nolint:gosec // our own template output
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		// Headers already sent — log only.
		_ = err
	}
}

// renderPartial renders a named template directly to the response (htmx swaps).
func (s *Server) renderPartial(w http.ResponseWriter, tmplName string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, tmplName, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
