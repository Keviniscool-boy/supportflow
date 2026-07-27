package testfixture

import "time"

const (
	TenantID        = "00000000-0000-0000-0000-000000000001"
	CustomerID      = "00000000-0000-0000-0000-000000000003"
	SessionID       = "00000000-0000-0000-0000-000000000002"
	OrderID         = "00000000-0000-0000-0000-000000000006"
	OrderNumber     = "SF20260001"
	DocumentID      = "00000000-0000-0000-0000-000000000012"
	DocumentVersion = "00000000-0000-0000-0000-000000000015"
	RawSessionToken = "supportflow-repeatable-test-session-token"
)

var FrozenTime = time.Date(2026, time.July, 25, 10, 0, 0, 0, time.UTC)

type Dataset struct {
	Tenant   Tenant
	Customer Customer
	Session  Session
	Order    Order
	Document Document
}

type Tenant struct {
	ID          string
	Slug        string
	DisplayName string
}

type Customer struct {
	ID          string
	TenantID    string
	DisplayName string
	Locale      string
}

type Session struct {
	ID         string
	TenantID   string
	CustomerID string
	RawToken   string
	ExpiresAt  time.Time
}

type Order struct {
	ID         string
	TenantID   string
	CustomerID string
	Number     string
	Status     string
}

type Document struct {
	ID        string
	VersionID string
	TenantID  string
	Title     string
	Content   string
}

func Standard() Dataset {
	return Dataset{
		Tenant:   Tenant{ID: TenantID, Slug: "novatech-test", DisplayName: "NovaTech 测试空间"},
		Customer: Customer{ID: CustomerID, TenantID: TenantID, DisplayName: "测试客户", Locale: "zh-CN"},
		Session:  Session{ID: SessionID, TenantID: TenantID, CustomerID: CustomerID, RawToken: RawSessionToken, ExpiresAt: FrozenTime.Add(time.Hour)},
		Order:    Order{ID: OrderID, TenantID: TenantID, CustomerID: CustomerID, Number: OrderNumber, Status: "PAID"},
		Document: Document{ID: DocumentID, VersionID: DocumentVersion, TenantID: TenantID, Title: "蓝牙耳机常见问题", Content: "重置耳机后重新连接蓝牙。"},
	}
}
