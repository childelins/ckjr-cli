package router

import "testing"

func TestInferRouteName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/admin/aiCreationCenter/modifyApp", "update"},
		{"/admin/aiCreationCenter/listApp", "listApp"},
		{"/admin/aiCreationCenter/createApp", "create"},
		{"/admin/aiCreationCenter/deleteApp", "deleteApp"},
		{"/admin/aiCreationCenter/describeApp", "get"},
		{"/admin/order/addOrder", "create"},
		{"/admin/order/removeOrder", "delete"},
		{"/admin/order/editOrder", "update"},
		{"/admin/order/queryList", "list"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := InferRouteName(tt.path)
			if got != tt.want {
				t.Errorf("InferRouteName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestInferNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"agent.yaml", "agent"},
		{"order.yaml", "order"},
		{"sub/dir/test.yaml", "test"},
		{"noext", "noext"},
		{"", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := InferNameFromPath(tt.path)
			if got != tt.want {
				t.Errorf("InferNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
