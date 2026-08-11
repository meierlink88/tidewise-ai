package rawdocument

import "testing"

func TestBusinessOperationsReturnsFreshRegistry(t *testing.T) {
	first := BusinessOperations()
	first[0] = "changed"
	if BusinessOperations()[0] != OperationList {
		t.Fatal("BusinessOperations returned mutable shared state")
	}
}
