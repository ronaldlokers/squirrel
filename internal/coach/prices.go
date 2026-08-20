package coach

// What a model costs, so the budget can be counted in money rather than in
// tokens.
//
// Tokens are the wrong unit for a ceiling. The ceiling is "about ten euros a
// month", and two models whose token counts are identical can differ tenfold in
// what they cost — so counting tokens would mean a budget that means something
// different depending on which model answered.
//
// The table is in code rather than in configuration on purpose: a price change
// should show up in a diff and be reviewed, not arrive silently through an
// environment variable nobody remembers setting. It was read from OpenAI's
// pricing page on 20 August 2026, and the model ids were confirmed against
// GET /v1/models on the same day rather than copied from prose.
//
// Cents per million tokens, as integers. Dollars are treated as euros for
// budgeting, which overstates the spend by whatever the rate is — erring
// toward stopping early is the right direction for a ceiling.

type price struct{ in, out int }

var prices = map[string]price{
	// The routine tier. Beats the previous generation's flagship on agentic
	// benchmarks; its documented weakness is long-context recall, which is why
	// nothing here hands it a large context.
	"gpt-5.6-luna": {in: 20, out: 120},
	// The escalation tier, for the turns where judgement matters.
	"gpt-5.6-terra": {in: 200, out: 1200},
	// Not used by default. Present so that pointing the config at it does not
	// silently cost nothing in the budget's opinion.
	"gpt-5.6-sol": {in: 500, out: 3000},
}

// Cost is what one answer cost, in **micro-euros** — millionths of a euro.
//
// The unit matters. A single routine answer costs well under a tenth of a
// cent, so counting in cents would round almost every call to zero and a
// month of them would report a spend of nothing at all. Micro-euros keep the
// arithmetic in integers with room to spare.
//
// An unknown model costs zero, which is a deliberate choice with a stated
// consequence: pointing the configuration at a model this table does not know
// means the budget stops protecting you. Boot warns about exactly that rather
// than leaving it silent.
func Cost(model string, in, out int) int64 {
	p, ok := prices[model]
	if !ok {
		return 0
	}
	// tokens × (cents per million tokens) = centi-cents; ÷100 → micro-euros.
	return (int64(in)*int64(p.in) + int64(out)*int64(p.out)) / 100
}

// KnownModel reports whether the budget can price this model. Boot calls it so
// a typo in configuration is a warning at start rather than a surprise at the
// end of the month.
func KnownModel(model string) bool {
	_, ok := prices[model]
	return ok
}
