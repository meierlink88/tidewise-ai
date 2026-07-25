package domain

import "testing"

func TestChainNodeAndThemeProfiles(t *testing.T) {
	tests := []struct {
		name    string
		profile interface{ Validate() error }
		wantErr bool
	}{
		{name: "chain node", profile: ChainNodeProfile{EntityID: "node", Definition: "可独立链接的产业概念"}},
		{name: "chain node optional boundary", profile: ChainNodeProfile{EntityID: "node", Definition: "可独立链接的产业概念", BoundaryNote: "不含行情标签"}},
		{name: "chain node blank definition", profile: ChainNodeProfile{EntityID: "node", Definition: " "}, wantErr: true},
		{name: "chain node blank optional boundary", profile: ChainNodeProfile{EntityID: "node", Definition: "节点", BoundaryNote: " "}, wantErr: true},
		{name: "theme", profile: ThemeProfile{EntityID: "theme", Definition: "自有投研视角", BoundaryNote: "不等同于产业链节点"}},
		{name: "theme missing boundary", profile: ThemeProfile{EntityID: "theme", Definition: "自有投研视角"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.profile.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestEntityNodeAcceptsTheme(t *testing.T) {
	node := EntityNode{ID: "theme", EntityType: EntityTypeTheme, LayerCode: "theme", Name: "主题", CanonicalName: "主题", Status: StatusActive}
	if err := node.Validate(); err != nil {
		t.Fatalf("EntityNode.Validate() error = %v", err)
	}
}
