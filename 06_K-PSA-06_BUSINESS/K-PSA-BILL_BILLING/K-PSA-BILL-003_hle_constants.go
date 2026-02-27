package psa

import "math"

// Headline License Equivalent (HLE) constants and pricing.
// One HLE = 10 agent seats OR 50M events OR 100,000 ML calls (max dimension wins).

const (
	HLEListPriceUSD     = 1500.0  // USD per HLE per month
	AgentsPerHLE        = 10      // agent seats per HLE unit
	EventsPerHLEMillion = 50      // events per HLE (in millions)
	MLCallsPerHLE       = 100_000 // ML API calls per HLE
)

// Tier defines a pricing tier band and its associated volume discount.
type Tier struct {
	MinHLE      float64
	MaxHLE      float64
	DiscountPct float64
}

// PricingTiers defines tiered discounts for HLE volume.
// MaxHLE of math.MaxFloat64 means no upper bound for that tier.
var PricingTiers = []Tier{
	{MinHLE: 0, MaxHLE: 5, DiscountPct: 0},
	{MinHLE: 5, MaxHLE: 10, DiscountPct: 5},
	{MinHLE: 10, MaxHLE: 20, DiscountPct: 10},
	{MinHLE: 20, MaxHLE: 50, DiscountPct: 15},
	{MinHLE: 50, MaxHLE: math.MaxFloat64, DiscountPct: 20},
}

// HLEBreakdown provides a complete view of dimension HLE values and final pricing.
type HLEBreakdown struct {
	AgentHLE    float64
	EventHLE    float64
	MLHLE       float64
	Total       float64
	DiscountPct float64
	FinalUSD    float64
}

// CalculateHLE returns the total HLE units using the highest-utilisation dimension.
func CalculateHLE(agentSeats int, eventsMillions float64, mlCalls int) float64 {
	agentHLE := float64(agentSeats) / float64(AgentsPerHLE)
	eventHLE := eventsMillions / float64(EventsPerHLEMillion)
	mlHLE := float64(mlCalls) / float64(MLCallsPerHLE)

	max := agentHLE
	if eventHLE > max {
		max = eventHLE
	}
	if mlHLE > max {
		max = mlHLE
	}
	return max
}

// discountForHLE returns the discount percentage for a given HLE total by searching tiers.
func discountForHLE(hle float64) float64 {
	for i := len(PricingTiers) - 1; i >= 0; i-- {
		t := PricingTiers[i]
		if hle >= t.MinHLE {
			return t.DiscountPct
		}
	}
	return 0
}

// ApplyTieredDiscount returns the net cost after applying the appropriate volume discount.
func ApplyTieredDiscount(hle float64) float64 {
	discount := discountForHLE(hle)
	gross := hle * HLEListPriceUSD
	return gross * (1 - discount/100)
}

// MonthlyCharge returns the after-discount monthly charge in USD for the given HLE units.
func MonthlyCharge(hle float64) float64 {
	return ApplyTieredDiscount(hle)
}

// CalculateWithBreakdown returns a full HLE breakdown including per-dimension values and final USD.
func CalculateWithBreakdown(seats int, eventsM float64, mlCalls int) HLEBreakdown {
	agentHLE := float64(seats) / float64(AgentsPerHLE)
	eventHLE := eventsM / float64(EventsPerHLEMillion)
	mlHLE := float64(mlCalls) / float64(MLCallsPerHLE)

	total := agentHLE
	if eventHLE > total {
		total = eventHLE
	}
	if mlHLE > total {
		total = mlHLE
	}

	disc := discountForHLE(total)
	gross := total * HLEListPriceUSD
	final := gross * (1 - disc/100)

	return HLEBreakdown{
		AgentHLE:    agentHLE,
		EventHLE:    eventHLE,
		MLHLE:       mlHLE,
		Total:       total,
		DiscountPct: disc,
		FinalUSD:    final,
	}
}
