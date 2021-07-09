package jetstream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/jsm.go"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta1"
	typed "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/typed/jetstream/v1beta1"
	"github.com/nats-io/nats.go"

	jsmapi "github.com/nats-io/jsm.go/api"

	k8sapi "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	streamTemplateFinalizerKey = "streamtemplatefinalizer.jetstream.nats.io"
)

func (c *Controller) runStreamTemplateQueue() {
	for {
		processQueueNext(c.strTmplQueue, &realJsmClient{jm: c.jm}, c.processStreamTemplate)
	}
}

func (c *Controller) processStreamTemplate(ns, name string, jsmc jsmClient) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to process stream template: %w", err)
		}
	}()

	strTmpl, err := c.strTmplLister.StreamTemplates(ns).Get(name)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	spec := strTmpl.Spec
	ifc := c.ji.StreamTemplates(strTmpl.Namespace)

	defer func() {
		if err == nil {
			return
		}

		if _, serr := setStreamTemplateErrored(c.ctx, strTmpl, ifc, err); serr != nil {
			err = fmt.Errorf("%s: %w", err, serr)
		}
	}()

	type operator func(ctx context.Context, c jsmClient, spec apis.StreamTemplateSpec) (err error)

	natsClientUtil := func(op operator) error {
		servers := spec.Servers
		if len(servers) != 0 {
			// Create a new client
			opts := make([]nats.Option, 0)
			opts = append(opts, nats.Name(fmt.Sprintf("%s-strtmpl-%s-%d", c.opts.NATSClientName, spec.Name, strTmpl.Generation)))
			// Use JWT/NKEYS based credentials if present.
			if spec.Creds != "" {
				opts = append(opts, nats.UserCredentials(spec.Creds))
			} else if spec.Nkey != "" {
				opt, err := nats.NkeyOptionFromSeed(spec.Nkey)
				if err != nil {
					return err
				}
				opts = append(opts, opt)
			}
			opts = append(opts, nats.MaxReconnects(-1))

			natsServers := strings.Join(servers, ",")
			newNc, err := nats.Connect(natsServers, opts...)
			if err != nil {
				return fmt.Errorf("failed to connect to nats-servers(%s): %w", natsServers, err)
			}

			c.normalEvent(strTmpl, "Connecting", "Connecting to new nats-servers")
			newJm, err := jsm.New(newNc)
			if err != nil {
				return err
			}
			newJsmc := &realJsmClient{nc: newNc, jm: newJm}

			if err := op(c.ctx, newJsmc, spec); err != nil {
				return err
			}
			newJsmc.Close()
		} else {
			if err := op(c.ctx, jsmc, spec); err != nil {
				return err
			}
		}
		return nil
	}

	deleteOK := strTmpl.GetDeletionTimestamp() != nil
	newGeneration := strTmpl.Generation != strTmpl.Status.ObservedGeneration
	strTmplOK := true
	err = natsClientUtil(streamTemplateExists)
	var apierr jsmapi.ApiError
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		strTmplOK = false
	} else if err != nil {
		return err
	}
	updateOK := (strTmplOK && !deleteOK && newGeneration)
	createOK := (!strTmplOK && !deleteOK && newGeneration)

	switch {
	case createOK:
		c.normalEvent(strTmpl, "Creating", fmt.Sprintf("Creating stream template %q", spec.Name))
		if err := natsClientUtil(createStreamTemplate); err != nil {
			return err
		}

		res, err := setStreamTemplateFinalizer(c.ctx, strTmpl, ifc)
		if err != nil {
			return err
		}
		strTmpl = res

		if _, err := setStreamTemplateOK(c.ctx, strTmpl, ifc); err != nil {
			return err
		}
		c.normalEvent(strTmpl, "Created", fmt.Sprintf("Created stream template %q", spec.Name))
	case updateOK:
		c.warningEvent(strTmpl, "Updating", fmt.Sprintf("Stream template (%q) updates are not allowed, recreate to update", spec.Name))
	case deleteOK:
		c.normalEvent(strTmpl, "Deleting", fmt.Sprintf("Deleting stream template %q", spec.Name))
		if err := natsClientUtil(deleteStreamTemplate); err != nil {
			return err
		}

		if _, err := clearStreamTemplateFinalizer(c.ctx, strTmpl, ifc); err != nil {
			return err
		}
	default:
		c.warningEvent(strTmpl, "Noop", fmt.Sprintf("Nothing done for stream template %q", spec.Name))
	}

	return nil
}

func streamTemplateExists(ctx context.Context, c jsmClient, spec apis.StreamTemplateSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to check if stream exists: %w", err)
		}
	}()

	_, err = c.LoadStreamTemplate(ctx, spec.Name)
	return err
}

func createStreamTemplate(ctx context.Context, c jsmClient, spec apis.StreamTemplateSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create stream template %q: %w", spec.Name, err)
		}
	}()

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

	_, err = c.NewStreamTemplate(ctx, jsmapi.StreamTemplateConfig{
		Name:       spec.Name,
		MaxStreams: uint32(spec.MaxStreams),
		Config: &jsmapi.StreamConfig{
			Retention:    retention,
			Subjects:     spec.Subjects,
			MaxConsumers: spec.MaxConsumers,
			MaxMsgs:      int64(spec.MaxMsgs),
			MaxBytes:     int64(spec.MaxBytes),
			MaxAge:       maxAge,
			MaxMsgSize:   int32(spec.MaxMsgSize),
			Storage:      storage,
			Discard:      discard,
			Replicas:     spec.Replicas,
			NoAck:        spec.NoAck,
			Template:     spec.Name,
			Duplicates:   duplicates,
		},
	})
	return err
}

func deleteStreamTemplate(ctx context.Context, c jsmClient, spec apis.StreamTemplateSpec) (err error) {
	name := spec.Name
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to delete stream template %q: %w", name, err)
		}
	}()

	var apierr jsmapi.ApiError
	st, err := c.LoadStreamTemplate(ctx, name)
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return nil
	} else if err != nil {
		return err
	}

	return st.Delete()
}

func setStreamTemplateOK(ctx context.Context, s *apis.StreamTemplate, i typed.StreamTemplateInterface) (*apis.StreamTemplate, error) {
	sc := s.DeepCopy()

	sc.Status.ObservedGeneration = s.Generation
	sc.Status.Conditions = upsertCondition(sc.Status.Conditions, apis.Condition{
		Type:               readyCondType,
		Status:             k8sapi.ConditionTrue,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Created",
		Message:            "Stream Template successfully created",
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set stream %q status: %w", s.Spec.Name, err)
	}

	return res, nil
}

func setStreamTemplateErrored(ctx context.Context, s *apis.StreamTemplate, i typed.StreamTemplateInterface, err error) (*apis.StreamTemplate, error) {
	if err == nil {
		return s, nil
	}

	sc := s.DeepCopy()
	sc.Status.Conditions = upsertCondition(sc.Status.Conditions, apis.Condition{
		Type:               readyCondType,
		Status:             k8sapi.ConditionFalse,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Errored",
		Message:            err.Error(),
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set stream errored status: %w", err)
	}

	return res, nil
}

func setStreamTemplateFinalizer(ctx context.Context, s *apis.StreamTemplate, i typed.StreamTemplateInterface) (*apis.StreamTemplate, error) {
	s.SetFinalizers(addFinalizer(s.GetFinalizers(), streamTemplateFinalizerKey))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.Update(ctx, s, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set finalizers for stream template %q: %w", s.Spec.Name, err)
	}

	return res, nil
}

func clearStreamTemplateFinalizer(ctx context.Context, s *apis.StreamTemplate, i typed.StreamTemplateInterface) (*apis.StreamTemplate, error) {
	s.SetFinalizers(removeFinalizer(s.GetFinalizers(), streamTemplateFinalizerKey))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.Update(ctx, s, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to clear finalizers for stream template %q: %w", s.Spec.Name, err)
	}

	return res, nil
}
