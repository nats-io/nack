package controller

import (
	"testing"
	"time"

	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNatsConfigFromOptsRejectsInvalidAccountAuth(t *testing.T) {
	authSecret := &api.SecretRef{Name: "account-auth"}
	tests := []struct {
		name       string
		spec       api.AccountSpec
		secretData map[string][]byte
		wantErr    string
	}{
		{
			name: "missing credentials secret key",
			spec: api.AccountSpec{Creds: &api.CredsSecret{
				File:   "user.creds",
				Secret: authSecret,
			}},
			secretData: map[string][]byte{},
			wantErr:    `account "test-account" credentials key "user.creds" not found in secret "account-auth"`,
		},
		{
			name: "empty credentials secret value",
			spec: api.AccountSpec{Creds: &api.CredsSecret{
				File:   "user.creds",
				Secret: authSecret,
			}},
			secretData: map[string][]byte{"user.creds": {}},
			wantErr:    `account "test-account" credentials key "user.creds" in secret "account-auth" is empty`,
		},
		{
			name: "whitespace-only credentials secret value",
			spec: api.AccountSpec{Creds: &api.CredsSecret{
				File:   "user.creds",
				Secret: authSecret,
			}},
			secretData: map[string][]byte{"user.creds": []byte(" \n\t")},
			wantErr:    `account "test-account" credentials key "user.creds" in secret "account-auth" is empty`,
		},
		{
			name:    "empty credentials file",
			spec:    api.AccountSpec{Creds: &api.CredsSecret{}},
			wantErr: `account "test-account" credentials file is empty`,
		},
		{
			name:    "missing nkey secret reference",
			spec:    api.AccountSpec{NKey: &api.NKeySecret{Seed: "seed"}},
			wantErr: `account "test-account" nkey secret is required`,
		},
		{
			name: "missing nkey seed key",
			spec: api.AccountSpec{NKey: &api.NKeySecret{
				Seed:   "seed",
				Secret: authSecret,
			}},
			secretData: map[string][]byte{},
			wantErr:    `account "test-account" nkey seed key "seed" not found in secret "account-auth"`,
		},
		{
			name: "empty nkey seed",
			spec: api.AccountSpec{NKey: &api.NKeySecret{
				Seed:   "seed",
				Secret: authSecret,
			}},
			secretData: map[string][]byte{"seed": {}},
			wantErr:    `account "test-account" nkey seed key "seed" in secret "account-auth" is empty`,
		},
		{
			name: "missing token key",
			spec: api.AccountSpec{Token: &api.TokenSecret{
				Token:  "token",
				Secret: *authSecret,
			}},
			secretData: map[string][]byte{},
			wantErr:    `account "test-account" token key "token" not found in secret "account-auth"`,
		},
		{
			name: "empty token",
			spec: api.AccountSpec{Token: &api.TokenSecret{
				Token:  "token",
				Secret: *authSecret,
			}},
			secretData: map[string][]byte{"token": {}},
			wantErr:    `account "test-account" token key "token" in secret "account-auth" is empty`,
		},
		{
			name: "missing username key",
			spec: api.AccountSpec{User: &api.User{
				User:     "username",
				Password: "password",
				Secret:   *authSecret,
			}},
			secretData: map[string][]byte{"password": []byte("password")},
			wantErr:    `account "test-account" username key "username" not found in secret "account-auth"`,
		},
		{
			name: "empty username",
			spec: api.AccountSpec{User: &api.User{
				User:     "username",
				Password: "password",
				Secret:   *authSecret,
			}},
			secretData: map[string][]byte{"username": {}, "password": []byte("password")},
			wantErr:    `account "test-account" username key "username" in secret "account-auth" is empty`,
		},
		{
			name: "missing password key",
			spec: api.AccountSpec{User: &api.User{
				User:     "username",
				Password: "password",
				Secret:   *authSecret,
			}},
			secretData: map[string][]byte{"username": []byte("user")},
			wantErr:    `account "test-account" password key "password" not found in secret "account-auth"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, api.AddToScheme(scheme))
			require.NoError(t, v1.AddToScheme(scheme))

			account := &api.Account{
				ObjectMeta: metav1.ObjectMeta{Name: "test-account", Namespace: "default"},
				Spec:       tt.spec,
			}
			objects := []client.Object{account}
			if tt.secretData != nil {
				objects = append(objects, &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: authSecret.Name, Namespace: "default"},
					Data:       tt.secretData,
				})
			}

			controller := &jsController{
				Client:   fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build(),
				config:   &NatsConfig{Credentials: "global.creds"},
				cacheDir: t.TempDir(),
			}

			config, err := controller.natsConfigFromOpts(api.ConnectionOpts{Account: account.Name}, account.Namespace)

			assert.Nil(t, config)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestNatsConfigFromOptsAllowsEmptyAccountPassword(t *testing.T) {
	tests := []struct {
		name        string
		passwordKey string
		secretData  map[string][]byte
	}{
		{
			name:       "omitted password selector",
			secretData: map[string][]byte{"username": []byte("user")},
		},
		{
			name:        "empty password secret value",
			passwordKey: "password",
			secretData:  map[string][]byte{"username": []byte("user"), "password": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, api.AddToScheme(scheme))
			require.NoError(t, v1.AddToScheme(scheme))

			account := &api.Account{
				ObjectMeta: metav1.ObjectMeta{Name: "test-account", Namespace: "default"},
				Spec: api.AccountSpec{User: &api.User{
					User:     "username",
					Password: tt.passwordKey,
					Secret:   api.SecretRef{Name: "account-auth"},
				}},
			}
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "account-auth", Namespace: "default"},
				Data:       tt.secretData,
			}
			controller := &jsController{
				Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(account, secret).Build(),
				config: &NatsConfig{
					ServerURL:   "nats://localhost:4222",
					Credentials: "global.creds",
				},
				cacheDir: t.TempDir(),
			}

			config, err := controller.natsConfigFromOpts(api.ConnectionOpts{Account: account.Name}, account.Namespace)

			require.NoError(t, err)
			assert.Empty(t, config.Credentials)
			assert.Equal(t, "user", config.User)
			assert.Empty(t, config.Password)
			assert.True(t, config.HasAuth())
			options, err := config.buildOptions()
			require.NoError(t, err)
			assert.Len(t, options, 1)
		})
	}
}

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
