// Copyright 2020 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jetstream

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	jsm "github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/api"
	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	typed "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/typed/jetstream/v1beta2"
	k8sapi "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	klog "k8s.io/klog/v2"
)

func (c *Controller) runStreamQueue() {
	for {
		processQueueNext(c.strQueue, c.RealJSMC, c.processStream)
	}
}

func (c *Controller) processStream(ns, name string, jsm jsmClientFunc) (err error) {
	str, err := c.strLister.Streams(ns).Get(name)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	return c.processStreamObject(str, jsm)
}

func (c *Controller) processStreamObject(str *apis.Stream, jsm jsmClientFunc) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to process stream: %w", err)
		}
	}()

	spec := str.Spec
	ifc := c.ji.Streams(str.Namespace)
	ns := str.Namespace
	readOnly := c.opts.ReadOnly

	var (
		remoteClientCert string
		remoteClientKey  string
		remoteRootCA     string
		accServers       []string
		acc              *apis.Account
		accUserCreds     string
	)
	if spec.Account != "" && c.opts.CRDConnect {
		// Lookup the account using the REST client.
		ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		acc, err = c.ji.Accounts(ns).Get(ctx, spec.Account, k8smeta.GetOptions{})
		if err != nil {
			return err
		}

		accServers = acc.Spec.Servers

		// Lookup the TLS secrets
		if acc.Spec.TLS != nil && acc.Spec.TLS.Secret != nil {
			secretName := acc.Spec.TLS.Secret.Name
			secret, err := c.ki.Secrets(ns).Get(c.ctx, secretName, k8smeta.GetOptions{})
			if err != nil {
				return err
			}

			// Write this to the cacheDir.
			accDir := filepath.Join(c.cacheDir, ns, spec.Account)
			if err := os.MkdirAll(accDir, 0o755); err != nil {
				return err
			}

			remoteClientCert = filepath.Join(accDir, acc.Spec.TLS.ClientCert)
			remoteClientKey = filepath.Join(accDir, acc.Spec.TLS.ClientKey)
			remoteRootCA = filepath.Join(accDir, acc.Spec.TLS.RootCAs)

			for k, v := range secret.Data {
				if err := os.WriteFile(filepath.Join(accDir, k), v, 0o644); err != nil {
					return err
				}
			}
		}
		// Lookup the UserCredentials.
		if acc.Spec.Creds != nil {
			secretName := acc.Spec.Creds.Secret.Name
			secret, err := c.ki.Secrets(ns).Get(c.ctx, secretName, k8smeta.GetOptions{})
			if err != nil {
				return err
			}

			// Write the user credentials to the cache dir.
			accDir := filepath.Join(c.cacheDir, ns, spec.Account)
			if err := os.MkdirAll(accDir, 0o755); err != nil {
				return err
			}
			for k, v := range secret.Data {
				if k == acc.Spec.Creds.File {
					accUserCreds = filepath.Join(c.cacheDir, ns, spec.Account, k)
					if err := os.WriteFile(filepath.Join(accDir, k), v, 0o644); err != nil {
						return err
					}
				}
			}
		}
	}

	defer func() {
		if err == nil {
			return
		}

		if _, serr := setStreamErrored(c.ctx, str, ifc, err); serr != nil {
			err = fmt.Errorf("%s: %w", err, serr)
		}
	}()

	type operator func(ctx context.Context, c jsmClient, spec apis.StreamSpec) (err error)

	natsClientUtil := func(op operator) error {
		servers := spec.Servers
		if c.opts.CRDConnect {
			// Create a new client
			natsCtx := &natsContext{}
			// Use JWT/NKEYS based credentials if present.
			if spec.Creds != "" {
				natsCtx.Credentials = spec.Creds
			} else if spec.Nkey != "" {
				natsCtx.Nkey = spec.Nkey
			}
			if spec.TLS.ClientCert != "" && spec.TLS.ClientKey != "" {
				natsCtx.TLSCert = spec.TLS.ClientCert
				natsCtx.TLSKey = spec.TLS.ClientKey
			}

			// Use fetched secrets for the account and server if defined.
			if remoteClientCert != "" && remoteClientKey != "" {
				natsCtx.TLSCert = remoteClientCert
				natsCtx.TLSKey = remoteClientKey
			}
			if remoteRootCA != "" {
				natsCtx.TLSCAs = []string{remoteRootCA}
			}
			if accUserCreds != "" {
				natsCtx.Credentials = accUserCreds
			}
			if len(spec.TLS.RootCAs) > 0 {
				natsCtx.TLSCAs = spec.TLS.RootCAs
			}

			natsServers := strings.Join(append(servers, accServers...), ",")
			natsCtx.URL = natsServers
			c.normalEvent(str, "Connecting", "Connecting to new nats-servers")
			jsmc, err := jsm(natsCtx)
			if err != nil {
				return fmt.Errorf("failed to connect to nats-servers(%s): %w", natsServers, err)
			}
			defer jsmc.Close()
			if err := op(c.ctx, jsmc, spec); err != nil {
				return err
			}
		} else {
			jsmc, err := jsm(&natsContext{})
			if err != nil {
				return err
			}
			if err := op(c.ctx, jsmc, spec); err != nil {
				return err
			}
		}
		return nil
	}

	deleteOK := str.GetDeletionTimestamp() != nil
	newGeneration := str.Generation != str.Status.ObservedGeneration
	strOK := true
	err = natsClientUtil(streamExists)
	var apierr jsmapi.ApiError
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		strOK = false
	} else if err != nil {
		return err
	}
	updateOK := (strOK && !deleteOK && newGeneration)
	createOK := (!strOK && !deleteOK) || (!updateOK && !deleteOK && newGeneration)

	switch {
	case createOK:
		if readOnly {
			c.normalEvent(str, "SkipCreate", fmt.Sprintf("Skip creating stream %q", spec.Name))
			return nil
		}
		c.normalEvent(str, "Creating", fmt.Sprintf("Creating stream %q", spec.Name))
		if err := natsClientUtil(createStream); err != nil {
			return err
		}

		if _, err := setStreamOK(c.ctx, str, ifc); err != nil {
			return err
		}
		c.normalEvent(str, "Created", fmt.Sprintf("Created stream %q", spec.Name))
	case updateOK:
		if str.Spec.PreventUpdate || readOnly {
			c.normalEvent(str, "SkipUpdate", fmt.Sprintf("Skip updating stream %q", spec.Name))
			if _, err := setStreamOK(c.ctx, str, ifc); err != nil {
				return err
			}
			return nil
		}
		c.normalEvent(str, "Updating", fmt.Sprintf("Updating stream %q", spec.Name))
		if err := natsClientUtil(updateStream); err != nil {
			return err
		}

		if _, err := setStreamOK(c.ctx, str, ifc); err != nil {
			return err
		}
		c.normalEvent(str, "Updated", fmt.Sprintf("Updated stream %q", spec.Name))
		return nil
	case deleteOK:
		if str.Spec.PreventDelete || readOnly {
			c.normalEvent(str, "SkipDelete", fmt.Sprintf("Skip deleting stream %q", spec.Name))
			if _, err := setStreamOK(c.ctx, str, ifc); err != nil {
				return err
			}
			return nil
		}
		c.normalEvent(str, "Deleting", fmt.Sprintf("Deleting stream %q", spec.Name))
		if err := natsClientUtil(deleteStream); err != nil {
			return err
		}
	default:
		c.normalEvent(str, "Noop", fmt.Sprintf("Nothing done for stream %q (prevent-delete=%v, prevent-update=%v)",
			spec.Name, spec.PreventDelete, spec.PreventUpdate,
		))
		// Noop events only update the status of the CRD.
		if _, err := setStreamOK(c.ctx, str, ifc); err != nil {
			return err
		}
	}

	return nil
}

func streamExists(ctx context.Context, c jsmClient, spec apis.StreamSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to check if stream exists: %w", err)
		}
	}()

	_, err = c.LoadStream(ctx, spec.Name)
	return err
}

func createStream(ctx context.Context, c jsmClient, spec apis.StreamSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create stream %q: %w", spec.Name, err)
		}
	}()

	maxAge, err := getMaxAge(spec.MaxAge)
	if err != nil {
		return err
	}

	duplicates, err := getDuplicates(spec.DuplicateWindow)
	if err != nil {
		return err
	}

	opts := []jsm.StreamOption{
		jsm.Subjects(spec.Subjects...),
		jsm.MaxConsumers(spec.MaxConsumers),
		jsm.MaxMessageSize(int32(spec.MaxMsgSize)),
		jsm.MaxMessages(int64(spec.MaxMsgs)),
		jsm.Replicas(spec.Replicas),
		jsm.DuplicateWindow(duplicates),
		jsm.MaxAge(maxAge),
		jsm.MaxBytes(int64(spec.MaxBytes)),
	}

	switch spec.Retention {
	case "limits":
		opts = append(opts, jsm.LimitsRetention())
	case "interest":
		opts = append(opts, jsm.InterestRetention())
	case "workqueue":
		opts = append(opts, jsm.WorkQueueRetention())
	}

	switch spec.Storage {
	case "file":
		opts = append(opts, jsm.FileStorage())
	case "memory":
		opts = append(opts, jsm.MemoryStorage())
	}

	switch spec.Discard {
	case "old":
		opts = append(opts, jsm.DiscardOld())
	case "new":
		opts = append(opts, jsm.DiscardNew())
	}

	switch spec.Compression {
	case "s2":
		opts = append(opts, jsm.Compression(api.S2Compression))
	case "none":
		opts = append(opts, jsm.Compression(api.NoCompression))
	}

	if spec.NoAck {
		opts = append(opts, jsm.NoAck())
	}

	if spec.Description != "" {
		opts = append(opts, func(o *jsmapi.StreamConfig) error {
			o.Description = spec.Description
			return nil
		})
	}

	if spec.MaxMsgsPerSubject > 0 {
		opts = append(opts, func(o *jsmapi.StreamConfig) error {
			o.MaxMsgsPer = int64(spec.MaxMsgsPerSubject)
			return nil
		})
	}

	if spec.Mirror != nil {
		ss, err := getStreamSource(spec.Mirror)
		if err != nil {
			return err
		}

		opts = append(opts, func(o *jsmapi.StreamConfig) error {
			o.Mirror = ss
			return nil
		})
	}

	if spec.Placement != nil {
		opts = append(opts, func(o *jsmapi.StreamConfig) error {
			o.Placement = &jsmapi.Placement{
				Cluster: spec.Placement.Cluster,
				Tags:    spec.Placement.Tags,
			}
			return nil
		})
	}

	var srcs []*jsmapi.StreamSource
	for _, ss := range spec.Sources {
		jss, err := getStreamSource(ss)
		if err != nil {
			return err
		}
		srcs = append(srcs, jss)
	}
	opts = append(opts, func(o *jsmapi.StreamConfig) error {
		o.Sources = srcs
		return nil
	})

	if spec.Republish != nil {
		opts = append(opts, jsm.Republish(&jsmapi.RePublish{
			Source:      spec.Republish.Source,
			Destination: spec.Republish.Destination,
		}))
	}

	if spec.SubjectTransform != nil {
		opts = append(opts, func(o *api.StreamConfig) error {
			o.SubjectTransform = &jsmapi.SubjectTransformConfig{
				Source:      spec.SubjectTransform.Source,
				Destination: spec.SubjectTransform.Dest,
			}
			return nil
		})
	}

	if spec.AllowDirect {
		opts = append(opts, jsm.AllowDirect())
	}

	if spec.AllowRollup {
		opts = append(opts, jsm.AllowRollup())
	}

	if spec.DenyDelete {
		opts = append(opts, jsm.DenyDelete())
	}

	if spec.DenyPurge {
		opts = append(opts, jsm.DenyPurge())
	}

	if spec.DiscardPerSubject {
		opts = append(opts, jsm.DiscardNewPerSubject())
	}

	if spec.FirstSequence != 0 {
		opts = append(opts, jsm.FirstSequence(spec.FirstSequence))
	}

	if spec.Metadata != nil {
		opts = append(opts, jsm.StreamMetadata(spec.Metadata))
	}

	_, err = c.NewStream(ctx, spec.Name, opts)
	return err
}

func updateStream(ctx context.Context, c jsmClient, spec apis.StreamSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to update stream %q: %w", spec.Name, err)
		}
	}()

	js, err := c.LoadStream(ctx, spec.Name)
	if err != nil {
		return err
	}

	maxAge, err := getMaxAge(spec.MaxAge)
	if err != nil {
		return err
	}

	retention := getRetention(spec.Retention)
	storage := getStorage(spec.Storage)
	discard := getDiscard(spec.Discard)

	duplicates, err := getDuplicates(spec.DuplicateWindow)
	if err != nil {
		return err
	}

	var subjectTransform *jsmapi.SubjectTransformConfig
	if spec.SubjectTransform != nil {
		subjectTransform = &jsmapi.SubjectTransformConfig{
			Source:      spec.SubjectTransform.Source,
			Destination: spec.SubjectTransform.Dest,
		}
	}

	config := jsmapi.StreamConfig{
		Name:             spec.Name,
		Description:      spec.Description,
		Retention:        retention,
		Subjects:         spec.Subjects,
		MaxConsumers:     spec.MaxConsumers,
		MaxMsgs:          int64(spec.MaxMsgs),
		MaxBytes:         int64(spec.MaxBytes),
		MaxMsgsPer:       int64(spec.MaxMsgsPerSubject),
		MaxAge:           maxAge,
		MaxMsgSize:       int32(spec.MaxMsgSize),
		Storage:          storage,
		Discard:          discard,
		DiscardNewPer:    spec.DiscardPerSubject,
		Replicas:         spec.Replicas,
		NoAck:            spec.NoAck,
		Duplicates:       duplicates,
		AllowDirect:      spec.AllowDirect,
		DenyDelete:       spec.DenyDelete,
		DenyPurge:        spec.DenyPurge,
		RollupAllowed:    spec.AllowRollup,
		FirstSeq:         spec.FirstSequence,
		SubjectTransform: subjectTransform,
	}
	if spec.Republish != nil {
		config.RePublish = &jsmapi.RePublish{
			Source:      spec.Republish.Source,
			Destination: spec.Republish.Destination,
			HeadersOnly: spec.Republish.HeadersOnly,
		}
	}
	if spec.Mirror != nil {
		ss, err := getStreamSource(spec.Mirror)
		if err != nil {
			return err
		}

		config.Mirror = ss
	}
	config.Sources = make([]*jsmapi.StreamSource, len(spec.Sources))
	for i, ss := range spec.Sources {
		jss, err := getStreamSource(ss)
		if err != nil {
			return err
		}
		config.Sources[i] = jss
	}

	if spec.Placement != nil {
		config.Placement = &jsmapi.Placement{
			Cluster: spec.Placement.Cluster,
			Tags:    spec.Placement.Tags,
		}
	}

	if spec.Metadata != nil {
		config.Metadata = spec.Metadata
	}

	switch spec.Compression {
	case "s2":
		config.Compression = api.S2Compression
	case "none":
		config.Compression = api.NoCompression
	}

	return js.UpdateConfiguration(config)
}

func deleteStream(ctx context.Context, c jsmClient, spec apis.StreamSpec) (err error) {
	name := spec.Name
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to delete stream %q: %w", name, err)
		}
	}()

	if spec.PreventDelete {
		klog.Infof("Stream %q is configured to preventDelete:\n", name)
		return nil
	}

	var apierr jsmapi.ApiError
	str, err := c.LoadStream(ctx, name)
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return nil
	} else if err != nil {
		return err
	}

	return str.Delete()
}

func setStreamErrored(ctx context.Context, s *apis.Stream, sif typed.StreamInterface, err error) (*apis.Stream, error) {
	if err == nil {
		return s, nil
	}

	sc := s.DeepCopy()
	sc.Status.Conditions = UpsertCondition(sc.Status.Conditions, apis.Condition{
		Type:               readyCondType,
		Status:             k8sapi.ConditionFalse,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Errored",
		Message:            err.Error(),
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var res *apis.Stream
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		res, err = sif.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to set stream errored status: %w", err)
		}
		return nil
	})
	return res, err
}

func setStreamOK(ctx context.Context, s *apis.Stream, i typed.StreamInterface) (*apis.Stream, error) {
	sc := s.DeepCopy()

	sc.Status.ObservedGeneration = s.Generation
	sc.Status.Conditions = UpsertCondition(sc.Status.Conditions, apis.Condition{
		Type:               readyCondType,
		Status:             k8sapi.ConditionTrue,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Created",
		Message:            "Stream successfully created",
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var res *apis.Stream
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		res, err = i.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to set stream %q status: %w", s.Spec.Name, err)
		}
		return nil
	})
	return res, err
}

func getMaxAge(v string) (time.Duration, error) {
	if v == "" {
		return time.Duration(0), nil
	}

	return time.ParseDuration(v)
}

func getRetention(v string) jsmapi.RetentionPolicy {
	retention := jsmapi.LimitsPolicy
	switch v {
	case "interest":
		retention = jsmapi.InterestPolicy
	case "workqueue":
		retention = jsmapi.WorkQueuePolicy
	}
	return retention
}

func getStorage(v string) jsmapi.StorageType {
	storage := jsmapi.MemoryStorage
	switch v {
	case "file":
		storage = jsmapi.FileStorage
	}
	return storage
}

func getDiscard(v string) jsmapi.DiscardPolicy {
	discard := jsmapi.DiscardOld
	switch v {
	case "new":
		discard = jsmapi.DiscardNew
	}
	return discard
}

func getDuplicates(v string) (time.Duration, error) {
	if v == "" {
		return time.Duration(0), nil
	}

	return time.ParseDuration(v)
}

func getStreamSource(ss *apis.StreamSource) (*jsmapi.StreamSource, error) {
	jss := &jsmapi.StreamSource{
		Name:          ss.Name,
		FilterSubject: ss.FilterSubject,
	}

	if ss.OptStartSeq > 0 {
		jss.OptStartSeq = uint64(ss.OptStartSeq)
	} else if ss.OptStartTime != "" {
		t, err := time.Parse(time.RFC3339, ss.OptStartTime)
		if err != nil {
			return nil, err
		}
		jss.OptStartTime = &t
	}

	if ss.ExternalAPIPrefix != "" || ss.ExternalDeliverPrefix != "" {
		jss.External = &jsmapi.ExternalStream{
			ApiPrefix:     ss.ExternalAPIPrefix,
			DeliverPrefix: ss.ExternalDeliverPrefix,
		}
	}

	for _, transform := range ss.SubjectTransforms {
		jss.SubjectTransforms = append(jss.SubjectTransforms, jsmapi.SubjectTransformConfig{
			Source:      transform.Source,
			Destination: transform.Dest,
		})
	}

	return jss, nil
}
