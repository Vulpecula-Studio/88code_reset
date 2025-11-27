package scheduler

import (
	"time"

	"code88reset/internal/models"
	"code88reset/internal/storage"
	"code88reset/pkg/logger"
)

type accountUpdater interface {
	UpdateGlobal(sub *models.Subscription)
	UpdateByEmail(email string, sub *models.Subscription)
}

type storageAccountUpdater struct {
	storage *storage.Storage
}

func newAccountUpdater(store *storage.Storage) accountUpdater {
	if store == nil {
		return noopAccountUpdater{}
	}
	return storageAccountUpdater{storage: store}
}

func (u storageAccountUpdater) UpdateGlobal(sub *models.Subscription) {
	if sub == nil {
		return
	}
	account := &models.AccountInfo{
		EmployeeID:         sub.EmployeeID,
		EmployeeName:       sub.EmployeeName,
		EmployeeEmail:      sub.EmployeeEmail,
		FreeSubscriptionID: sub.ID,
		CurrentCredits:     sub.CurrentCredits,
		CreditLimit:        sub.SubscriptionPlan.CreditLimit,
		ResetTimes:         sub.ResetTimes,
		LastCreditReset:    sub.LastCreditReset,
	}

	if err := u.storage.SaveAccountInfo(account); err != nil {
		logger.Error("保存账号信息失败: %v", err)
	} else {
		logger.Debug("账号信息已更新")
	}
}

func (u storageAccountUpdater) UpdateByEmail(email string, sub *models.Subscription) {
	if sub == nil || email == "" {
		return
	}
	accountInfo := &models.AccountInfo{
		EmployeeID:         sub.EmployeeID,
		EmployeeName:       sub.EmployeeName,
		EmployeeEmail:      sub.EmployeeEmail,
		FreeSubscriptionID: sub.ID,
		CurrentCredits:     sub.CurrentCredits,
		CreditLimit:        sub.SubscriptionPlan.CreditLimit,
		ResetTimes:         sub.ResetTimes,
		LastCreditReset:    sub.LastCreditReset,
		LastUpdated:        time.Now(),
	}

	if err := u.storage.SaveAccountInfoByEmail(email, accountInfo); err != nil {
		logger.Warn("保存账号信息失败 (Email=%s): %v", email, err)
	}
}

type noopAccountUpdater struct{}

func (noopAccountUpdater) UpdateGlobal(*models.Subscription)          {}
func (noopAccountUpdater) UpdateByEmail(string, *models.Subscription) {}
