package domain

import "regexp"

// SymptomRule maps error patterns to likely failure layers.
type SymptomRule struct {
	Pattern     string   // regex pattern to match symptom text
	Layer       string   // "backend-auth", "backend-routing", "backend-bug", "frontend", "infra", "data"
	Description string   // human-readable explanation
	CheckOrder  []string // node kinds to check first, e.g. ["service", "repository"]
	Confidence  float64  // 0.0-1.0 how diagnostic this pattern is (1.0 = almost certainly this layer)
}

// DefaultSymptomRules returns the built-in symptom matching rules.
// Rules are ordered from most specific to least specific within each
// category. MatchSymptom returns all matches ranked by confidence.
func DefaultSymptomRules() []SymptomRule {
	return []SymptomRule{
		// --- High confidence: specific error patterns ---
		{
			Pattern:     `(?i)(unique constraint|duplicate key|duplicate entry|violates unique)`,
			Layer:       "data",
			Description: "Duplicate record: insert or update violates a uniqueness constraint",
			CheckOrder:  []string{"repository", "query", "service"},
			Confidence:  0.95,
		},
		{
			Pattern:     `(?i)(foreign key.*violat|referential integrity|fk constraint)`,
			Layer:       "data",
			Description: "Referential integrity violation: referenced record does not exist or is being deleted",
			CheckOrder:  []string{"repository", "query", "service"},
			Confidence:  0.95,
		},
		{
			Pattern:     `(?i)(deadlock detected|deadlock found)`,
			Layer:       "data",
			Description: "Database deadlock: concurrent transactions competing for the same rows",
			CheckOrder:  []string{"repository", "service"},
			Confidence:  0.95,
		},
		{
			Pattern:     `(?i)(sql:\s*no rows|no rows in result|record not found)`,
			Layer:       "data",
			Description: "Query returned no rows: expected record does not exist",
			CheckOrder:  []string{"repository", "query", "service"},
			Confidence:  0.90,
		},
		{
			Pattern:     `(?i)(json:\s*cannot unmarshal|json.*unmarshal|invalid character.*looking for)`,
			Layer:       "backend-bug",
			Description: "JSON deserialization failure: response or request body does not match expected structure",
			CheckOrder:  []string{"handler", "service", "external"},
			Confidence:  0.90,
		},
		{
			Pattern:     `(?i)(login failed|invalid credentials|wrong password|authentication error)`,
			Layer:       "backend-auth",
			Description: "Login flow failure: credentials rejected or auth service unavailable",
			CheckOrder:  []string{"service", "handler", "external"},
			Confidence:  0.90,
		},
		{
			Pattern:     `(?i)(connection refused|ECONNREFUSED|dial tcp|network unreachable|ENETUNREACH)`,
			Layer:       "infra",
			Description: "Service unreachable: target server is down or network path is broken",
			CheckOrder:  []string{"external", "repository"},
			Confidence:  0.90,
		},
		{
			Pattern:     `(?i)(CORS|origin not allowed|cross-origin|access-control-allow-origin)`,
			Layer:       "infra",
			Description: "Cross-origin request blocked: CORS policy misconfiguration",
			CheckOrder:  []string{"handler", "external"},
			Confidence:  0.90,
		},
		{
			Pattern:     `(?i)(selector.*not found|element not found|no element|getBy.*failed|queryBy.*null)`,
			Layer:       "frontend",
			Description: "UI element missing: component not rendered or selector has changed",
			CheckOrder:  []string{"component", "hook"},
			Confidence:  0.85,
		},
		{
			Pattern:     `(?i)(TypeError|Cannot read propert|undefined is not|is not a function)`,
			Layer:       "frontend",
			Description: "JavaScript runtime error: accessing property on undefined/null or calling non-function",
			CheckOrder:  []string{"component", "hook", "service"},
			Confidence:  0.85,
		},
		{
			Pattern:     `(?i)(hydration.*mismatch|hydration.*failed|text content does not match)`,
			Layer:       "frontend",
			Description: "SSR hydration mismatch: server-rendered HTML differs from client render",
			CheckOrder:  []string{"component", "hook"},
			Confidence:  0.85,
		},

		// --- Medium-high confidence: HTTP status codes with context ---
		{
			Pattern:     `(?i)(no route|route not matched|route not found|endpoint.*not found)`,
			Layer:       "backend-routing",
			Description: "Route not registered: the HTTP endpoint does not exist in the router",
			CheckOrder:  []string{"endpoint", "handler"},
			Confidence:  0.85,
		},
		{
			Pattern:     `(?i)(409|conflict)`,
			Layer:       "data",
			Description: "Resource conflict: duplicate resource or optimistic locking failure",
			CheckOrder:  []string{"repository", "service", "handler"},
			Confidence:  0.80,
		},
		{
			Pattern:     `(?i)(422|unprocessable|validation failed|invalid.*field|missing.*required)`,
			Layer:       "backend-bug",
			Description: "Validation failure: request data does not meet business rules",
			CheckOrder:  []string{"handler", "service"},
			Confidence:  0.80,
		},
		{
			Pattern:     `(?i)(429|rate.?limit|too many requests|throttl)`,
			Layer:       "infra",
			Description: "Rate limit exceeded: too many requests to the service or upstream",
			CheckOrder:  []string{"handler", "external"},
			Confidence:  0.80,
		},
		{
			Pattern:     `(?i)(502|bad gateway)`,
			Layer:       "infra",
			Description: "Upstream service error: reverse proxy received an invalid response",
			CheckOrder:  []string{"external", "handler"},
			Confidence:  0.80,
		},
		{
			Pattern:     `(?i)(503|service unavailable)`,
			Layer:       "infra",
			Description: "Service unavailable: server overloaded, in maintenance, or dependency down",
			CheckOrder:  []string{"external", "service"},
			Confidence:  0.80,
		},
		{
			Pattern:     `(?i)(context canceled|request canceled|client disconnected)`,
			Layer:       "infra",
			Description: "Request canceled: client disconnected or upstream context was canceled",
			CheckOrder:  []string{"service", "external"},
			Confidence:  0.75,
		},
		{
			Pattern:     `(?i)(EOF|unexpected EOF|broken pipe|connection reset)`,
			Layer:       "infra",
			Description: "Connection dropped: remote end closed the connection mid-transfer",
			CheckOrder:  []string{"external", "repository", "service"},
			Confidence:  0.75,
		},
		{
			Pattern:     `(?i)(tls|certificate|x509|cert.*expir|ssl)`,
			Layer:       "infra",
			Description: "TLS/certificate error: invalid, expired, or untrusted certificate",
			CheckOrder:  []string{"external"},
			Confidence:  0.85,
		},

		// --- Medium confidence: status codes that are ambiguous ---
		{
			Pattern:     `(?i)(401|unauthorized|unauthenticated)`,
			Layer:       "backend-auth",
			Description: "Authentication failure: request lacks valid credentials or session has expired",
			CheckOrder:  []string{"handler", "service", "external"},
			Confidence:  0.70,
		},
		{
			Pattern:     `(?i)(403|forbidden|permission denied|access denied)`,
			Layer:       "backend-auth",
			Description: "Authorization failure: authenticated user lacks required permissions",
			CheckOrder:  []string{"handler", "service"},
			Confidence:  0.70,
		},
		{
			Pattern:     `(?i)(404|not found)`,
			Layer:       "backend-routing",
			Description: "Not found: endpoint does not exist or requested resource is missing",
			CheckOrder:  []string{"endpoint", "handler", "repository"},
			Confidence:  0.55,
		},
		{
			Pattern:     `(?i)(timeout|timed? ?out|deadline exceeded|context deadline|ETIMEDOUT)`,
			Layer:       "infra",
			Description: "Operation exceeded time limit: network, database, or service latency",
			CheckOrder:  []string{"external", "repository", "service"},
			Confidence:  0.60,
		},

		// --- Lower confidence: broad patterns ---
		{
			Pattern:     `(?i)(500|internal server error|panic|runtime error|nil pointer)`,
			Layer:       "backend-bug",
			Description: "Server-side crash or unhandled error in business logic",
			CheckOrder:  []string{"service", "repository", "handler"},
			Confidence:  0.50,
		},
		{
			Pattern:     `(?i)(empty response|no data|null response|empty body|content.length.*0)`,
			Layer:       "data",
			Description: "Response contained no data: query returned nothing or serialization failed",
			CheckOrder:  []string{"repository", "query", "service"},
			Confidence:  0.50,
		},
	}
}

// MatchSymptom returns all matching rules for a symptom string, ordered by
// confidence (highest first). Returns nil if no rule matches.
func MatchSymptom(symptom string, rules []SymptomRule) []*SymptomRule {
	var matches []*SymptomRule
	for i := range rules {
		re, err := regexp.Compile(rules[i].Pattern)
		if err != nil {
			continue
		}
		if re.MatchString(symptom) {
			matches = append(matches, &rules[i])
		}
	}

	// Sort by confidence descending. Use insertion sort since the list is small.
	for i := 1; i < len(matches); i++ {
		for j := i; j > 0 && matches[j].Confidence > matches[j-1].Confidence; j-- {
			matches[j], matches[j-1] = matches[j-1], matches[j]
		}
	}

	return matches
}
