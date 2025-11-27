package reset

import (
	"strings"
	"testing"

	"code88reset/internal/models"
)

func TestShouldSkipByThreshold_FirstResetUsesMaxThreshold(t *testing.T) {
	r := &Runner{
		opts: Options{
			ResetType:          "first",
			UseMaxThreshold:    true,
			CreditThresholdMax: 80,
		},
	}

	sub := models.Subscription{
		CurrentCredits: 90,
		SubscriptionPlan: models.SubscriptionPlan{
			CreditLimit: 100,
		},
	}

	skip, reason := r.shouldSkipByThreshold(sub)
	if !skip {
		t.Fatalf("expected skip=true for first reset over threshold, got false")
	}
	if !strings.Contains(reason, "额度充足") {
		t.Fatalf("expected skip reason to mention额度充足, got %q", reason)
	}
}

func TestShouldSkipByThreshold_SecondResetIgnoresThreshold(t *testing.T) {
	r := &Runner{
		opts: Options{
			ResetType:          "second",
			UseMaxThreshold:    true,
			CreditThresholdMax: 80,
		},
	}

	sub := models.Subscription{
		CurrentCredits: 90,
		SubscriptionPlan: models.SubscriptionPlan{
			CreditLimit: 100,
		},
	}

	skip, reason := r.shouldSkipByThreshold(sub)
	if skip {
		t.Fatalf("expected skip=false for second reset, got true with reason %q", reason)
	}
	if reason != "" {
		t.Fatalf("expected empty reason for second reset, got %q", reason)
	}
}
