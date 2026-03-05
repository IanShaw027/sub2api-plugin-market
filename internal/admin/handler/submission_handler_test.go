package handler

import "testing"

func TestNormalizeReviewRequest(t *testing.T) {
	tests := []struct {
		name       string
		input      ReviewRequest
		wantAction string
		wantNotes  string
		wantErr    bool
	}{
		{
			name:       "兼容旧 action approved 与 comment",
			input:      ReviewRequest{Action: "approved", Comment: "通过"},
			wantAction: "approve",
			wantNotes:  "通过",
		},
		{
			name:       "兼容旧 action rejected 并去除空白",
			input:      ReviewRequest{Action: "  Rejected  ", Comment: "不通过"},
			wantAction: "reject",
			wantNotes:  "不通过",
		},
		{
			name:       "新字段保持不变且 reviewer_notes 优先",
			input:      ReviewRequest{Action: "approve", ReviewerNotes: "新意见", Comment: "旧意见"},
			wantAction: "approve",
			wantNotes:  "新意见",
		},
		{
			name:    "非法 action 返回错误",
			input:   ReviewRequest{Action: "pass"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.input
			err := normalizeReviewRequest(&req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if req.Action != tt.wantAction {
				t.Fatalf("action = %s, want = %s", req.Action, tt.wantAction)
			}

			if req.ReviewerNotes != tt.wantNotes {
				t.Fatalf("reviewer_notes = %s, want = %s", req.ReviewerNotes, tt.wantNotes)
			}
		})
	}
}
