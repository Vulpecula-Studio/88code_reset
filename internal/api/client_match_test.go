package api

import (
	"testing"

	"code88reset/internal/models"
)

func TestMatchesTargetPlan_DefaultMonthly(t *testing.T) {
	sub := models.Subscription{
		ID:               1,
		SubscriptionName: "PLUS",
		SubscriptionPlan: models.SubscriptionPlan{
			SubscriptionName: "PLUS",
			PlanType:         "MONTHLY",
		},
	}

	if !matchesTargetPlan(sub, nil) {
		t.Fatal("expected monthly plan to match when no targets configured")
	}
}

func TestMatchesTargetPlan_FilterBySubscriptionName(t *testing.T) {
	sub := models.Subscription{
		ID:               3,
		SubscriptionName: "PLUS",
		SubscriptionPlan: models.SubscriptionPlan{
			SubscriptionName: "PLUS",
			PlanType:         "MONTHLY",
		},
	}

	targets := buildTargetPlanSet([]string{"plus", " pro "})
	if !matchesTargetPlan(sub, targets) {
		t.Fatal("expected PLUS to match target set")
	}

	other := models.Subscription{
		ID:               4,
		SubscriptionName: "MAX",
		SubscriptionPlan: models.SubscriptionPlan{
			SubscriptionName: "MAX",
			PlanType:         "MONTHLY",
		},
	}
	if matchesTargetPlan(other, targets) {
		t.Fatal("did not expect MAX to match PLUS target set")
	}
}

func TestMatchesTargetPlan_SkipsNonMonthly(t *testing.T) {
	sub := models.Subscription{
		ID:               5,
		SubscriptionName: "PAYGO",
		SubscriptionPlan: models.SubscriptionPlan{
			SubscriptionName: "PAYGO",
			PlanType:         "PAY_PER_USE",
		},
	}

	if matchesTargetPlan(sub, nil) {
		t.Fatal("expected PAYGO plan to be skipped even without targets")
	}

	targets := buildTargetPlanSet([]string{"paygo"})
	if matchesTargetPlan(sub, targets) {
		t.Fatal("expected PAYGO plan to be skipped even when explicitly targeted")
	}
}
