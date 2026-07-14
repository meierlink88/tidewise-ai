package domain

import "testing"

func TestEntityExternalIdentifierValidation(t *testing.T) {
	valid := EntityExternalIdentifier{
		ID:                 "identifier-id",
		EntityID:           "entity-id",
		SourceSystem:       "eastmoney",
		SourceTaxonomyType: "concept_sector",
		ExternalCode:       "BK0619",
		ExternalName:       "3D打印",
		Status:             StatusActive,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	tests := []EntityExternalIdentifier{
		{EntityID: valid.EntityID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: valid.ExternalCode, ExternalName: valid.ExternalName, Status: valid.Status},
		{ID: valid.ID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: valid.ExternalCode, ExternalName: valid.ExternalName, Status: valid.Status},
		{ID: valid.ID, EntityID: valid.EntityID, SourceSystem: " ", SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: valid.ExternalCode, ExternalName: valid.ExternalName, Status: valid.Status},
		{ID: valid.ID, EntityID: valid.EntityID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: " ", ExternalCode: valid.ExternalCode, ExternalName: valid.ExternalName, Status: valid.Status},
		{ID: valid.ID, EntityID: valid.EntityID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: " ", ExternalName: valid.ExternalName, Status: valid.Status},
		{ID: valid.ID, EntityID: valid.EntityID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: valid.ExternalCode, ExternalName: " ", Status: valid.Status},
		{ID: valid.ID, EntityID: valid.EntityID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: valid.ExternalCode, ExternalName: valid.ExternalName, Status: StatusMerged},
	}
	for i, candidate := range tests {
		if err := candidate.Validate(); err == nil {
			t.Fatalf("case %d Validate() error = nil", i)
		}
	}
}
