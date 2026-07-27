package testfixture

import "testing"

func TestStandardDatasetIsRepeatable(t *testing.T) {
	first := Standard()
	second := Standard()
	if first != second {
		t.Fatalf("fixture must be repeatable: %#v != %#v", first, second)
	}
	if first.Order.CustomerID != first.Customer.ID || first.Session.CustomerID != first.Customer.ID {
		t.Fatal("fixture object ownership is inconsistent")
	}
}
