package negotiation

import "fmt"

// Template generates a human-readable counter-offer message for a round.
type Template struct{}

// CounterOfferMessage generates a templated message for a negotiation round.
func (t *Template) CounterOfferMessage(round int, discountPct float64, role string) string {
	switch role {
	case "buyer":
		return t.buyerMessage(round, discountPct)
	case "seller":
		return t.sellerMessage(round, discountPct)
	default:
		return fmt.Sprintf("Proposal at %.0f%% discount (round %d)", discountPct*100, round)
	}
}

func (t *Template) buyerMessage(round int, discountPct float64) string {
	messages := []string{
		fmt.Sprintf("We're interested but need better pricing. Can you do %.0f%% off list?", discountPct*100),
		fmt.Sprintf("We have competing offers. At %.0f%% discount we can move forward today.", discountPct*100),
		fmt.Sprintf("Our budget is constrained. We need at least %.0f%% to proceed.", discountPct*100),
		fmt.Sprintf("We're committing to a multi-year term. %.0f%% discount makes this work.", discountPct*100),
	}
	return messages[(round-1)%len(messages)]
}

func (t *Template) sellerMessage(round int, discountPct float64) string {
	messages := []string{
		fmt.Sprintf("We can offer %.0f%% discount for annual commitment.", discountPct*100),
		fmt.Sprintf("Approved %.0f%% discount — best we can do for your volume tier.", discountPct*100),
		fmt.Sprintf("Final offer at %.0f%% — includes onboarding and support.", discountPct*100),
		fmt.Sprintf("At %.0f%% we're stretching — but we value the partnership.", discountPct*100),
	}
	return messages[(round-1)%len(messages)]
}
