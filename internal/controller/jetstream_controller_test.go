package controller

import (
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
	"time"
)

func Test_updateReadyCondition(t *testing.T) {

	pastTransition := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano)
	updatedTransition := "now"

	otherCondition := api.Condition{
		Type:               "other",
		Status:             v1.ConditionFalse,
		Reason:             "Reason",
		Message:            "Message",
		LastTransitionTime: pastTransition,
	}

	type args struct {
		conditions []api.Condition
		status     v1.ConditionStatus
		reason     string
		message    string
	}
	tests := []struct {
		name string
		args args
		want []api.Condition
	}{
		{
			name: "new ready condition",
			args: args{
				conditions: nil,
				status:     v1.ConditionTrue,
				reason:     "Test",
				message:    "Test Message",
			},
			want: []api.Condition{
				{
					Type:               readyCondType,
					Status:             v1.ConditionTrue,
					Reason:             "Test",
					Message:            "Test Message",
					LastTransitionTime: updatedTransition,
				},
			},
		},
		{
			name: "update ready condition",
			args: args{
				conditions: []api.Condition{
					otherCondition,
					{
						Type:               readyCondType,
						Status:             v1.ConditionFalse,
						Reason:             "Test",
						Message:            "Test Message",
						LastTransitionTime: pastTransition,
					},
				},
				status:  v1.ConditionTrue,
				reason:  "New Reason",
				message: "New Message",
			},
			want: []api.Condition{
				otherCondition,
				{
					Type:               readyCondType,
					Status:             v1.ConditionTrue,
					Reason:             "New Reason",
					Message:            "New Message",
					LastTransitionTime: updatedTransition,
				},
			},
		},
		{
			name: "should not update transition time when status is not changed",
			args: args{
				conditions: []api.Condition{
					otherCondition,
					{
						Type:               readyCondType,
						Status:             v1.ConditionTrue,
						Reason:             "Test",
						Message:            "Test Message",
						LastTransitionTime: pastTransition,
					},
				},
				status:  v1.ConditionTrue,
				reason:  "New Reason",
				message: "New Message",
			},
			want: []api.Condition{
				otherCondition,
				{
					Type:               readyCondType,
					Status:             v1.ConditionTrue,
					Reason:             "New Reason",
					Message:            "New Message",
					LastTransitionTime: pastTransition,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got := updateReadyCondition(tt.args.conditions, tt.args.status, tt.args.reason, tt.args.message)

			assert.Len(got, len(tt.want))
			for i, want := range tt.want {
				actual := got[i]

				assert.Equal(actual.Type, want.Type)
				assert.Equal(actual.Status, want.Status)
				assert.Equal(actual.Reason, want.Reason)
				assert.Equal(actual.Message, want.Message)

				// Assert transition time was updated
				if want.LastTransitionTime == updatedTransition {
					actualTransitionTime, err := time.Parse(time.RFC3339Nano, actual.LastTransitionTime)
					assert.NoError(err)
					assert.WithinDuration(actualTransitionTime, time.Now(), 5*time.Second)
				}
				// Assert transition time was not updated
				if want.LastTransitionTime == pastTransition {
					assert.Equal(pastTransition, actual.LastTransitionTime)
				}
			}
		})
	}
}
