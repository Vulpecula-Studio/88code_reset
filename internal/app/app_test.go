package app

import (
	"testing"
	"time"

	appconfig "code88reset/internal/config"
	"code88reset/internal/models"
	"code88reset/internal/storage"
)

type fakeAccountManager struct {
	listAccountsResp []models.AccountConfig
	listErr          error
	listCalled       bool

	countTotal    int
	countEnabled  int
	countDisabled int
	countErr      error
	countCalled   bool

	syncCalled bool
	syncKeys   []string
	syncPlans  []string
	syncErr    error

	activeAccountsResp []models.AccountConfig
	activeErr          error
	activeCalled       bool
	activeKeys         []string
}

func (f *fakeAccountManager) ListAccounts() ([]models.AccountConfig, error) {
	f.listCalled = true
	return f.listAccountsResp, f.listErr
}

func (f *fakeAccountManager) GetAccountCount() (int, int, int, error) {
	f.countCalled = true
	return f.countTotal, f.countEnabled, f.countDisabled, f.countErr
}

func (f *fakeAccountManager) SyncAccountsFromAPIKeys(apiKeys []string, targetPlans []string) error {
	f.syncCalled = true
	f.syncKeys = append([]string(nil), apiKeys...)
	f.syncPlans = append([]string(nil), targetPlans...)
	return f.syncErr
}

func (f *fakeAccountManager) GetActiveAccountsFromAPIKeys(apiKeys []string) ([]models.AccountConfig, error) {
	f.activeCalled = true
	f.activeKeys = append([]string(nil), apiKeys...)
	return f.activeAccountsResp, f.activeErr
}

type fakeClient struct {
	subscriptions            []models.Subscription
	target                   *models.Subscription
	testConnectionCalled     bool
	getSubscriptionsCalled   bool
	getTargetSubscriptionCnt int
	resetCreditsCalled       bool
	lastResetID              int
	resetErr                 error
}

func (f *fakeClient) TestConnection() error {
	f.testConnectionCalled = true
	return nil
}

func (f *fakeClient) GetSubscriptions() ([]models.Subscription, error) {
	f.getSubscriptionsCalled = true
	return f.subscriptions, nil
}

func (f *fakeClient) GetTargetSubscription() (*models.Subscription, error) {
	f.getTargetSubscriptionCnt++
	return f.target, nil
}

func (f *fakeClient) ResetCredits(subscriptionID int) (*models.ResetResponse, error) {
	f.resetCreditsCalled = true
	f.lastResetID = subscriptionID
	if f.resetErr != nil {
		return nil, f.resetErr
	}
	return &models.ResetResponse{Success: true, Message: "ok"}, nil
}

func newTestApp(t *testing.T, cfg appconfig.Settings, mgr accountManager) *App {
	t.Helper()

	// storage is unused in tests but construct to satisfy default wiring if needed
	store, err := storage.NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	app := New(cfg, store, mgr)
	app.deps.sleep = func(time.Duration) {}
	return app
}

func TestAppRun_TestMode(t *testing.T) {
	cfg := appconfig.Settings{
		Mode:    "test",
		APIKeys: []string{"k1"},
		Plans:   []string{"FREE"},
	}

	mgr := &fakeAccountManager{}
	app := newTestApp(t, cfg, mgr)

	client := &fakeClient{
		subscriptions: []models.Subscription{
			{
				ID:                 1,
				SubscriptionName:   "FREE",
				SubscriptionPlan:   models.SubscriptionPlan{CreditLimit: 20},
				CurrentCredits:     10,
				ResetTimes:         2,
				SubscriptionPlanID: 1,
			},
		},
		target: &models.Subscription{
			ID:               1,
			SubscriptionName: "FREE",
			SubscriptionPlan: models.SubscriptionPlan{CreditLimit: 20, PlanType: "MONTHLY"},
			CurrentCredits:   5,
			ResetTimes:       3,
		},
	}

	app.deps.newClient = func(*storage.Storage, string, string, []string) apiClient {
		return client
	}
	app.deps.runSingleScheduler = func(*App, apiClient) error {
		t.Fatalf("scheduler should not be invoked in test mode")
		return nil
	}
	app.deps.runMultiScheduler = func(*App, []models.AccountConfig) error {
		t.Fatalf("multi scheduler should not be invoked in test mode")
		return nil
	}

	if err := app.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !client.testConnectionCalled || !client.getSubscriptionsCalled {
		t.Fatalf("client methods not called as expected: %+v", client)
	}
	if mgr.listCalled || mgr.syncCalled || mgr.activeCalled {
		t.Fatalf("account manager should not be used in test mode")
	}
}

func TestAppRun_RunModeSingle(t *testing.T) {
	cfg := appconfig.Settings{
		Mode:    "run",
		APIKeys: []string{"single"},
		Plans:   []string{"FREE"},
	}
	mgr := &fakeAccountManager{}
	app := newTestApp(t, cfg, mgr)

	client := &fakeClient{}
	app.deps.newClient = func(*storage.Storage, string, string, []string) apiClient { return client }

	called := false
	app.deps.runSingleScheduler = func(a *App, c apiClient) error {
		called = true
		if c != client {
			t.Fatalf("unexpected client passed to scheduler")
		}
		return nil
	}
	app.deps.runMultiScheduler = func(*App, []models.AccountConfig) error {
		t.Fatalf("multi scheduler should not be invoked for single run mode")
		return nil
	}

	if err := app.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !called {
		t.Fatalf("single scheduler was not invoked")
	}
}

func TestAppRun_RunModeMulti(t *testing.T) {
	cfg := appconfig.Settings{
		Mode:    "run",
		APIKeys: []string{"k1", "k2"},
		Plans:   []string{"FREE"},
		BaseURL: "https://example.com",
	}

	mgr := &fakeAccountManager{
		activeAccountsResp: []models.AccountConfig{
			{EmployeeEmail: "a@example.com"},
			{EmployeeEmail: "b@example.com"},
		},
		countTotal:    5,
		countEnabled:  4,
		countDisabled: 1,
	}
	app := newTestApp(t, cfg, mgr)

	client := &fakeClient{}
	app.deps.newClient = func(*storage.Storage, string, string, []string) apiClient { return client }
	app.deps.runSingleScheduler = func(*App, apiClient) error {
		t.Fatalf("single scheduler should not run in multi-account scenario")
		return nil
	}

	var received []models.AccountConfig
	app.deps.runMultiScheduler = func(a *App, accounts []models.AccountConfig) error {
		received = append([]models.AccountConfig(nil), accounts...)
		return nil
	}

	if err := app.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !mgr.syncCalled || !mgr.activeCalled || !mgr.countCalled {
		t.Fatalf("expected account manager interactions, got %+v", mgr)
	}
	if len(received) != len(mgr.activeAccountsResp) {
		t.Fatalf("expected %d accounts, got %d", len(mgr.activeAccountsResp), len(received))
	}
}
