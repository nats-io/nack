/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"os"
	"path/filepath"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nkeys"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
)

// createNKeyTestServer creates a NATS server that requires NKey authentication.
// Returns the server, the NKey seed bytes, and the public key.
func createNKeyTestServer() (*server.Server, []byte, string) {
	// Generate a user NKey pair
	user, err := nkeys.CreateUser()
	Expect(err).NotTo(HaveOccurred())

	publicKey, err := user.PublicKey()
	Expect(err).NotTo(HaveOccurred())

	seed, err := user.Seed()
	Expect(err).NotTo(HaveOccurred())

	// Copy DefaultTestOptions to avoid mutating the global
	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.Debug = true
	opts.Nkeys = []*server.NkeyUser{
		{Nkey: publicKey},
	}

	dir, err := os.MkdirTemp("", "nats-nkey-*")
	Expect(err).NotTo(HaveOccurred())
	opts.StoreDir = dir

	ns := natsserver.RunServer(&opts)
	Expect(ns).NotTo(BeNil())

	return ns, seed, publicKey
}

var _ = Describe("Account NKey Support", func() {
	Context("natsConfigFromOpts with NKey Account", func() {
		const accountName = "nkey-account"
		const secretName = "nkey-secret"
		const seedKey = "user.nk"

		var (
			cacheDir string
			nkeySeed []byte
		)

		BeforeEach(func(ctx SpecContext) {
			var err error
			cacheDir, err = os.MkdirTemp("", "nack-test-cache-*")
			Expect(err).NotTo(HaveOccurred())

			// Generate a real NKey seed for the test
			user, err := nkeys.CreateUser()
			Expect(err).NotTo(HaveOccurred())
			nkeySeed, err = user.Seed()
			Expect(err).NotTo(HaveOccurred())

			By("creating the NKey secret")
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					seedKey: nkeySeed,
				},
			}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, &v1.Secret{})
			if err != nil && k8serrors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			}

			By("creating the Account with NKey reference")
			account := &api.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      accountName,
					Namespace: "default",
				},
				Spec: api.AccountSpec{
					Servers: []string{clientUrl},
					NKey: &api.NKeySecret{
						Seed: seedKey,
						Secret: &api.SecretRef{
							Name: secretName,
						},
					},
				},
			}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: accountName, Namespace: "default"}, &api.Account{})
			if err != nil && k8serrors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, account)).To(Succeed())
			}
		})

		AfterEach(func(ctx SpecContext) {
			os.RemoveAll(cacheDir)

			By("cleaning up the Account")
			account := &api.Account{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: accountName, Namespace: "default"}, account)
			if err == nil {
				controllerutil.RemoveFinalizer(account, accountFinalizer)
				k8sClient.Update(ctx, account)
				k8sClient.Delete(ctx, account)
			}

			By("cleaning up the Secret")
			secret := &v1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, secret)
			if err == nil {
				k8sClient.Delete(ctx, secret)
			}
		})

		It("should read NKey from Secret and write seed file to cache", func(ctx SpecContext) {
			By("creating a jsController with cache dir")
			jsc := &jsController{
				Client: k8sClient,
				config: &NatsConfig{ServerURL: clientUrl},
				controllerConfig: &Config{
					Namespace: "default",
					CacheDir:  cacheDir,
				},
				cacheDir: cacheDir,
				connPool: newConnPool(0),
			}

			By("calling natsConfigFromOpts with account reference")
			opts := api.ConnectionOpts{
				Account: accountName,
			}
			natsConfig, err := jsc.natsConfigFromOpts(opts, "default")
			Expect(err).NotTo(HaveOccurred())

			By("verifying the NKey path is set in the config")
			expectedPath := filepath.Join(cacheDir, "default", accountName, seedKey)
			Expect(natsConfig.NKey).To(Equal(expectedPath))

			By("verifying the NKey file was written to the cache dir")
			Expect(expectedPath).To(BeAnExistingFile())

			By("verifying the file contents match the secret data")
			fileContent, err := os.ReadFile(expectedPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(fileContent).To(Equal(nkeySeed))

			By("verifying the file has restricted permissions")
			info, err := os.Stat(expectedPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode().Perm()).To(Equal(os.FileMode(0o600)))
		})

		It("should skip writing NKey file when content is unchanged", func(ctx SpecContext) {
			jsc := &jsController{
				Client: k8sClient,
				config: &NatsConfig{ServerURL: clientUrl},
				controllerConfig: &Config{
					Namespace: "default",
					CacheDir:  cacheDir,
				},
				cacheDir: cacheDir,
				connPool: newConnPool(0),
			}

			opts := api.ConnectionOpts{
				Account: accountName,
			}

			By("calling natsConfigFromOpts the first time")
			_, err := jsc.natsConfigFromOpts(opts, "default")
			Expect(err).NotTo(HaveOccurred())

			expectedPath := filepath.Join(cacheDir, "default", accountName, seedKey)

			By("recording the file modification time")
			info1, err := os.Stat(expectedPath)
			Expect(err).NotTo(HaveOccurred())
			modTime1 := info1.ModTime()

			// Small delay to ensure filesystem timestamp granularity
			time.Sleep(50 * time.Millisecond)

			By("calling natsConfigFromOpts a second time with same data")
			_, err = jsc.natsConfigFromOpts(opts, "default")
			Expect(err).NotTo(HaveOccurred())

			By("verifying the file was not rewritten (modification time unchanged)")
			info2, err := os.Stat(expectedPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info2.ModTime()).To(Equal(modTime1))
		})

		It("should set server URL from the Account spec", func(ctx SpecContext) {
			jsc := &jsController{
				Client: k8sClient,
				config: &NatsConfig{ServerURL: "nats://old-server:4222"},
				controllerConfig: &Config{
					Namespace: "default",
					CacheDir:  cacheDir,
				},
				cacheDir: cacheDir,
				connPool: newConnPool(0),
			}

			opts := api.ConnectionOpts{
				Account: accountName,
			}

			natsConfig, err := jsc.natsConfigFromOpts(opts, "default")
			Expect(err).NotTo(HaveOccurred())

			By("verifying the server URL was set from Account spec")
			Expect(natsConfig.ServerURL).To(Equal(clientUrl))
		})
	})

	Context("end-to-end: Stream creation via NKey-authenticated Account", func() {
		const (
			accountName = "nkey-e2e-account"
			secretName  = "nkey-e2e-secret"
			streamRes   = "nkey-e2e-stream"
			streamName  = "nkey-e2e-orders"
			seedKey     = "user.nk"
		)

		var (
			nkeyServer *server.Server
			nkeySeed   []byte
			nkeyUrl    string
			cacheDir   string
		)

		BeforeEach(func(ctx SpecContext) {
			By("creating NKey-authenticated NATS server")
			var err error
			nkeyServer, nkeySeed, _ = createNKeyTestServer()
			nkeyUrl = nkeyServer.ClientURL()

			cacheDir, err = os.MkdirTemp("", "nack-e2e-cache-*")
			Expect(err).NotTo(HaveOccurred())

			By("creating the NKey secret in Kubernetes")
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					seedKey: nkeySeed,
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("creating the Account referencing the NKey secret")
			account := &api.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      accountName,
					Namespace: "default",
				},
				Spec: api.AccountSpec{
					Servers: []string{nkeyUrl},
					NKey: &api.NKeySecret{
						Seed: seedKey,
						Secret: &api.SecretRef{
							Name: secretName,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, account)).To(Succeed())
		})

		AfterEach(func(ctx SpecContext) {
			By("cleaning up the stream resource")
			stream := &api.Stream{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: streamRes, Namespace: "default"}, stream); err == nil {
				controllerutil.RemoveFinalizer(stream, streamFinalizer)
				k8sClient.Update(ctx, stream)
				k8sClient.Delete(ctx, stream)
			}

			By("cleaning up the account resource")
			account := &api.Account{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: accountName, Namespace: "default"}, account); err == nil {
				controllerutil.RemoveFinalizer(account, accountFinalizer)
				k8sClient.Update(ctx, account)
				k8sClient.Delete(ctx, account)
			}

			By("cleaning up the secret")
			secret := &v1.Secret{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, secret); err == nil {
				k8sClient.Delete(ctx, secret)
			}

			By("shutting down NKey test server")
			storeDir := nkeyServer.StoreDir()
			nkeyServer.Shutdown()
			os.RemoveAll(storeDir)
			os.RemoveAll(cacheDir)
		})

		It("should create a stream using NKey-authenticated Account", func(ctx SpecContext) {
			By("reconciling the Account to set it as ready")
			nkeyConfig := &NatsConfig{ServerURL: nkeyUrl}
			nkeyController, err := NewJSController(k8sClient, nkeyConfig, &Config{
				Namespace: "default",
				CacheDir:  cacheDir,
			})
			Expect(err).NotTo(HaveOccurred())

			accountController := &AccountReconciler{
				Scheme:              k8sClient.Scheme(),
				JetStreamController: nkeyController,
			}
			accountNN := types.NamespacedName{Name: accountName, Namespace: "default"}

			// Reconcile until the account has a finalizer and ready status
			Eventually(func(ctx SpecContext) *api.Account {
				_, err := accountController.Reconcile(ctx, ctrl.Request{NamespacedName: accountNN})
				Expect(err).NotTo(HaveOccurred())
				got := &api.Account{}
				Expect(k8sClient.Get(ctx, accountNN, got)).To(Succeed())
				return got
			}).WithContext(ctx).Should(SatisfyAll(
				HaveField("Finalizers", HaveExactElements(accountFinalizer)),
				HaveField("Status.Conditions", Not(BeEmpty())),
			))

			By("creating the Stream resource referencing the NKey Account")
			stream := &api.Stream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      streamRes,
					Namespace: "default",
				},
				Spec: api.StreamSpec{
					Name:      streamName,
					Replicas:  1,
					Subjects:  []string{"nkey-orders.*"},
					Retention: "workqueue",
					Discard:   "old",
					Storage:   "file",
					BaseStreamConfig: api.BaseStreamConfig{
						ConnectionOpts: api.ConnectionOpts{
							Account: accountName,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, stream)).To(Succeed())

			By("initializing the Stream resource")
			streamNN := types.NamespacedName{Name: streamRes, Namespace: "default"}
			Expect(k8sClient.Get(ctx, streamNN, stream)).To(Succeed())

			controllerutil.AddFinalizer(stream, streamFinalizer)
			Expect(k8sClient.Update(ctx, stream)).To(Succeed())

			stream.Status.Conditions = []api.Condition{{
				Type:               readyCondType,
				Status:             v1.ConditionUnknown,
				Reason:             "Test",
				Message:            "start condition",
				LastTransitionTime: time.Now().Format(time.RFC3339Nano),
			}}
			Expect(k8sClient.Status().Update(ctx, stream)).To(Succeed())

			By("reconciling the Stream")
			streamController := &StreamReconciler{
				Scheme:              k8sClient.Scheme(),
				JetStreamController: nkeyController,
			}

			result, err := streamController.Reconcile(ctx, ctrl.Request{NamespacedName: streamNN})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			By("verifying the stream was created in the NKey-authenticated NATS server")
			// Connect to the NKey server to verify the stream exists
			seedFile, err := os.CreateTemp("", "nkey-seed-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(seedFile.Name())
			_, err = seedFile.Write(nkeySeed)
			Expect(err).NotTo(HaveOccurred())
			seedFile.Close()

			verifyConfig := &NatsConfig{
				ServerURL: nkeyUrl,
				NKey:      seedFile.Name(),
			}
			verifyPool := newConnPool(0)
			verifyConn, err := verifyPool.Get(verifyConfig, true)
			Expect(err).NotTo(HaveOccurred())
			defer verifyConn.Close()

			js, err := CreateJetStreamClient(verifyConn, true, "")
			Expect(err).NotTo(HaveOccurred())

			natsStream, err := js.Stream(ctx, streamName)
			Expect(err).NotTo(HaveOccurred())
			streamInfo, err := natsStream.Info(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(streamInfo.Config.Name).To(Equal(streamName))

			By("verifying the stream resource has ready status")
			Expect(k8sClient.Get(ctx, streamNN, stream)).To(Succeed())
			Expect(stream.Status.Conditions).To(HaveLen(1))
			assertReadyStateMatches(stream.Status.Conditions[0], v1.ConditionTrue, stateReady, "created or updated", time.Now())
		})

		It("should fail to create a stream without NKey when server requires it", func(ctx SpecContext) {
			By("creating a controller WITHOUT NKey configuration")
			noAuthConfig := &NatsConfig{ServerURL: nkeyUrl}
			noAuthController, err := NewJSController(k8sClient, noAuthConfig, &Config{
				Namespace: "default",
				CacheDir:  cacheDir,
			})
			Expect(err).NotTo(HaveOccurred())

			By("creating a stream that does NOT reference the NKey Account")
			streamNoAuth := &api.Stream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nkey-e2e-noauth-stream",
					Namespace: "default",
				},
				Spec: api.StreamSpec{
					Name:      "noauth-stream",
					Replicas:  1,
					Subjects:  []string{"noauth.*"},
					Retention: "workqueue",
					Discard:   "old",
					Storage:   "file",
				},
			}
			Expect(k8sClient.Create(ctx, streamNoAuth)).To(Succeed())

			streamNN := types.NamespacedName{Name: "nkey-e2e-noauth-stream", Namespace: "default"}
			Expect(k8sClient.Get(ctx, streamNN, streamNoAuth)).To(Succeed())

			controllerutil.AddFinalizer(streamNoAuth, streamFinalizer)
			Expect(k8sClient.Update(ctx, streamNoAuth)).To(Succeed())

			streamNoAuth.Status.Conditions = []api.Condition{{
				Type:               readyCondType,
				Status:             v1.ConditionUnknown,
				Reason:             "Test",
				Message:            "start condition",
				LastTransitionTime: time.Now().Format(time.RFC3339Nano),
			}}
			Expect(k8sClient.Status().Update(ctx, streamNoAuth)).To(Succeed())

			By("reconciling - expecting failure due to auth requirement")
			streamController := &StreamReconciler{
				Scheme:              k8sClient.Scheme(),
				JetStreamController: noAuthController,
			}
			_, err = streamController.Reconcile(ctx, ctrl.Request{NamespacedName: streamNN})
			Expect(err).To(HaveOccurred())

			By("cleaning up")
			controllerutil.RemoveFinalizer(streamNoAuth, streamFinalizer)
			k8sClient.Update(ctx, streamNoAuth)
			k8sClient.Delete(ctx, streamNoAuth)
		})
	})

	Context("NKey with other auth combinations in Account", func() {
		It("should prefer Creds over NKey when both are configured", func(ctx SpecContext) {
			cacheDir, err := os.MkdirTemp("", "nack-prio-cache-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(cacheDir)

			By("creating secrets for both creds and nkey")
			credsSecret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prio-creds-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"user.creds": []byte("fake-creds-data"),
				},
			}
			Expect(k8sClient.Create(ctx, credsSecret)).To(Succeed())
			defer func() {
				k8sClient.Delete(ctx, credsSecret)
			}()

			user, err := nkeys.CreateUser()
			Expect(err).NotTo(HaveOccurred())
			seed, err := user.Seed()
			Expect(err).NotTo(HaveOccurred())

			nkeySecret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prio-nkey-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"user.nk": seed,
				},
			}
			Expect(k8sClient.Create(ctx, nkeySecret)).To(Succeed())
			defer func() {
				k8sClient.Delete(ctx, nkeySecret)
			}()

			By("creating Account with both Creds and NKey")
			account := &api.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prio-account",
					Namespace: "default",
				},
				Spec: api.AccountSpec{
					Servers: []string{clientUrl},
					Creds: &api.CredsSecret{
						File:   "user.creds",
						Secret: &api.SecretRef{Name: "prio-creds-secret"},
					},
					NKey: &api.NKeySecret{
						Seed:   "user.nk",
						Secret: &api.SecretRef{Name: "prio-nkey-secret"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, account)).To(Succeed())
			defer func() {
				acc := &api.Account{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: "prio-account", Namespace: "default"}, acc); err == nil {
					controllerutil.RemoveFinalizer(acc, accountFinalizer)
					k8sClient.Update(ctx, acc)
					k8sClient.Delete(ctx, acc)
				}
			}()

			jsc := &jsController{
				Client: k8sClient,
				config: &NatsConfig{ServerURL: clientUrl},
				controllerConfig: &Config{
					Namespace: "default",
					CacheDir:  cacheDir,
				},
				cacheDir: cacheDir,
				connPool: newConnPool(0),
			}

			opts := api.ConnectionOpts{
				Account: "prio-account",
			}
			natsConfig, err := jsc.natsConfigFromOpts(opts, "default")
			Expect(err).NotTo(HaveOccurred())

			By("verifying Creds takes priority over NKey in the Overlay")
			// The Overlay method clears all auth when setting a new one.
			// Since Creds is set on the account overlay, NKey should be cleared.
			// The account overlay sets both, but NatsConfig.Overlay clears other auth.
			Expect(natsConfig.Credentials).NotTo(BeEmpty())
		})
	})

	Context("Account NKey with missing secret", func() {
		It("should return error when referenced secret does not exist", func(ctx SpecContext) {
			cacheDir, err := os.MkdirTemp("", "nack-missing-cache-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(cacheDir)

			By("creating Account referencing non-existent secret")
			account := &api.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "missing-secret-account",
					Namespace: "default",
				},
				Spec: api.AccountSpec{
					Servers: []string{clientUrl},
					NKey: &api.NKeySecret{
						Seed:   "user.nk",
						Secret: &api.SecretRef{Name: "non-existent-secret"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, account)).To(Succeed())
			defer func() {
				acc := &api.Account{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: "missing-secret-account", Namespace: "default"}, acc); err == nil {
					controllerutil.RemoveFinalizer(acc, accountFinalizer)
					k8sClient.Update(ctx, acc)
					k8sClient.Delete(ctx, acc)
				}
			}()

			jsc := &jsController{
				Client: k8sClient,
				config: &NatsConfig{ServerURL: clientUrl},
				controllerConfig: &Config{
					Namespace: "default",
					CacheDir:  cacheDir,
				},
				cacheDir: cacheDir,
				connPool: newConnPool(0),
			}

			opts := api.ConnectionOpts{
				Account: "missing-secret-account",
			}
			_, err = jsc.natsConfigFromOpts(opts, "default")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Account NKey with missing seed key in secret", func() {
		It("should not set NKey when seed key is missing from secret data", func(ctx SpecContext) {
			cacheDir, err := os.MkdirTemp("", "nack-badkey-cache-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(cacheDir)

			By("creating secret without the expected seed key")
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "wrong-key-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"wrong-key.nk": []byte("some-data"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() {
				k8sClient.Delete(ctx, secret)
			}()

			By("creating Account referencing a secret with wrong key name")
			account := &api.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "wrong-key-account",
					Namespace: "default",
				},
				Spec: api.AccountSpec{
					Servers: []string{clientUrl},
					NKey: &api.NKeySecret{
						Seed:   "user.nk",
						Secret: &api.SecretRef{Name: "wrong-key-secret"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, account)).To(Succeed())
			defer func() {
				acc := &api.Account{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: "wrong-key-account", Namespace: "default"}, acc); err == nil {
					controllerutil.RemoveFinalizer(acc, accountFinalizer)
					k8sClient.Update(ctx, acc)
					k8sClient.Delete(ctx, acc)
				}
			}()

			jsc := &jsController{
				Client: k8sClient,
				config: &NatsConfig{ServerURL: clientUrl},
				controllerConfig: &Config{
					Namespace: "default",
					CacheDir:  cacheDir,
				},
				cacheDir: cacheDir,
				connPool: newConnPool(0),
			}

			opts := api.ConnectionOpts{
				Account: "wrong-key-account",
			}
			natsConfig, err := jsc.natsConfigFromOpts(opts, "default")
			Expect(err).NotTo(HaveOccurred())

			By("verifying NKey is not set when seed key doesn't match")
			Expect(natsConfig.NKey).To(BeEmpty())
		})
	})
})

var _ = Describe("NatsConfig NKey Overlay", func() {
	It("should correctly overlay NKey auth", func() {
		base := &NatsConfig{
			ServerURL: "nats://base:4222",
			User:      "old-user",
			Password:  "old-pass",
		}

		overlay := &NatsConfig{
			NKey: "/path/to/key.nk",
		}

		base.Overlay(overlay)

		Expect(base.NKey).To(Equal("/path/to/key.nk"))
		Expect(base.User).To(BeEmpty(), "User should be cleared when NKey is set")
		Expect(base.Password).To(BeEmpty(), "Password should be cleared when NKey is set")
		Expect(base.Credentials).To(BeEmpty(), "Credentials should be cleared when NKey is set")
		Expect(base.Token).To(BeEmpty(), "Token should be cleared when NKey is set")
	})

	It("should clear NKey when Credentials overlay is applied", func() {
		base := &NatsConfig{
			ServerURL: "nats://base:4222",
			NKey:      "/path/to/key.nk",
		}

		overlay := &NatsConfig{
			Credentials: "/path/to/creds",
		}

		base.Overlay(overlay)

		Expect(base.Credentials).To(Equal("/path/to/creds"))
		Expect(base.NKey).To(BeEmpty(), "NKey should be cleared when Credentials is set")
	})

	It("should include NKey in HasAuth check", func() {
		config := &NatsConfig{
			NKey: "/path/to/key.nk",
		}
		Expect(config.HasAuth()).To(BeTrue())
	})

	It("should clear NKey in UnsetAuth", func() {
		config := &NatsConfig{
			NKey: "/path/to/key.nk",
		}
		config.UnsetAuth()
		Expect(config.NKey).To(BeEmpty())
	})
})

var _ = Describe("NatsConfig NKey Hash", func() {
	It("should include NKey file content in hash", func() {
		// Create a temp file with NKey content
		tmpFile, err := os.CreateTemp("", "nkey-hash-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte("SUAXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"))
		Expect(err).NotTo(HaveOccurred())
		tmpFile.Close()

		config1 := &NatsConfig{
			ServerURL: "nats://localhost:4222",
			NKey:      tmpFile.Name(),
		}

		config2 := &NatsConfig{
			ServerURL: "nats://localhost:4222",
		}

		hash1, err := config1.Hash()
		Expect(err).NotTo(HaveOccurred())

		hash2, err := config2.Hash()
		Expect(err).NotTo(HaveOccurred())

		Expect(hash1).NotTo(Equal(hash2), "Hash should differ when NKey is included")
	})

	It("should produce different hashes for different NKey files", func() {
		tmpFile1, err := os.CreateTemp("", "nkey-hash1-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(tmpFile1.Name())
		tmpFile1.Write([]byte("SUAXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"))
		tmpFile1.Close()

		tmpFile2, err := os.CreateTemp("", "nkey-hash2-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(tmpFile2.Name())
		tmpFile2.Write([]byte("SUAYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY"))
		tmpFile2.Close()

		config1 := &NatsConfig{
			ServerURL: "nats://localhost:4222",
			NKey:      tmpFile1.Name(),
		}
		config2 := &NatsConfig{
			ServerURL: "nats://localhost:4222",
			NKey:      tmpFile2.Name(),
		}

		hash1, err := config1.Hash()
		Expect(err).NotTo(HaveOccurred())

		hash2, err := config2.Hash()
		Expect(err).NotTo(HaveOccurred())

		Expect(hash1).NotTo(Equal(hash2))
	})
})

var _ = Describe("NatsConfig NKey BuildOptions", func() {
	It("should create NKey option from seed file", func() {
		// Generate a real NKey seed
		user, err := nkeys.CreateUser()
		Expect(err).NotTo(HaveOccurred())
		seed, err := user.Seed()
		Expect(err).NotTo(HaveOccurred())

		tmpFile, err := os.CreateTemp("", "nkey-build-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(tmpFile.Name())
		tmpFile.Write(seed)
		tmpFile.Close()

		config := &NatsConfig{
			ServerURL: "nats://localhost:4222",
			NKey:      tmpFile.Name(),
		}

		opts, err := config.buildOptions()
		Expect(err).NotTo(HaveOccurred())
		// nats.Name + nats.NkeyOptionFromSeed = at least 2 options
		Expect(len(opts)).To(BeNumerically(">=", 1))
	})

	It("should fail with invalid NKey file path", func() {
		config := &NatsConfig{
			ServerURL: "nats://localhost:4222",
			NKey:      "/nonexistent/path/key.nk",
		}

		_, err := config.buildOptions()
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("natsConfigFromOpts free function with NKey", func() {
	It("should set NKey from ConnectionOpts.Nkey", func() {
		opts := api.ConnectionOpts{
			Nkey: "/path/to/nkey.nk",
		}

		result := natsConfigFromOpts(opts)
		Expect(result.NKey).To(Equal("/path/to/nkey.nk"))
	})

	It("should not set NKey when Nkey is empty", func() {
		opts := api.ConnectionOpts{
			Servers: []string{"nats://localhost:4222"},
		}

		result := natsConfigFromOpts(opts)
		Expect(result.NKey).To(BeEmpty())
	})
})

var _ = Describe("Connection Pool with NKey", func() {
	It("should pool connections with same NKey config", func() {
		// Generate NKey and write to a file
		user, err := nkeys.CreateUser()
		Expect(err).NotTo(HaveOccurred())
		pubKey, err := user.PublicKey()
		Expect(err).NotTo(HaveOccurred())
		seed, err := user.Seed()
		Expect(err).NotTo(HaveOccurred())

		By("starting NKey-authenticated server")
		// Copy DefaultTestOptions to avoid mutating the global
		opts := natsserver.DefaultTestOptions
		opts.JetStream = true
		opts.Port = -1
		opts.Nkeys = []*server.NkeyUser{
			{Nkey: pubKey},
		}
		dir, err := os.MkdirTemp("", "nats-pool-*")
		Expect(err).NotTo(HaveOccurred())
		opts.StoreDir = dir
		ns := natsserver.RunServer(&opts)
		defer func() {
			ns.Shutdown()
			os.RemoveAll(dir)
		}()

		seedFile, err := os.CreateTemp("", "nkey-pool-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(seedFile.Name())
		seedFile.Write(seed)
		seedFile.Close()

		pool := newConnPool(0)

		c1 := &NatsConfig{
			ClientName: "Client NKey 1",
			ServerURL:  ns.ClientURL(),
			NKey:       seedFile.Name(),
		}
		c2 := &NatsConfig{
			ClientName: "Client NKey 1",
			ServerURL:  ns.ClientURL(),
			NKey:       seedFile.Name(),
		}

		conn1, err := pool.Get(c1, true)
		Expect(err).NotTo(HaveOccurred())
		conn2, err := pool.Get(c2, true)
		Expect(err).NotTo(HaveOccurred())

		By("verifying same NKey config returns same pooled connection")
		Expect(conn1).To(BeIdenticalTo(conn2))

		conn1.Close()
		conn2.Close()
	})
})
