package jetstream

import (
	"context"
	"errors"
	"fmt"
	"time"

	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta1"
	typed "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/typed/jetstream/v1beta1"

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
		processQueueNext(c.strTmplQueue, &realJsmClient{c.nc}, c.processStreamTemplate)
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

	deleteOK := strTmpl.GetDeletionTimestamp() != nil
	newGeneration := strTmpl.Generation != strTmpl.Status.ObservedGeneration
	strTmplOK, err := streamTemplateExists(c.ctx, jsmc, spec.Name)
	if err != nil {
		return err
	}
	updateOK := (strTmplOK && !deleteOK && newGeneration)
	createOK := (!strTmplOK && !deleteOK && newGeneration)

	switch {
	case createOK:
		c.normalEvent(strTmpl, "Creating", fmt.Sprintf("Creating stream template %q", spec.Name))
		if err := createStreamTemplate(c.ctx, jsmc, spec); err != nil {
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
		if err := deleteStreamTemplate(c.ctx, jsmc, spec.Name); err != nil {
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

func streamTemplateExists(ctx context.Context, c jsmClient, name string) (ok bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to check if stream exists: %w", err)
		}
	}()

	var apierr jsmapi.ApiError
	_, err = c.LoadStreamTemplate(ctx, name)
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
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

func deleteStreamTemplate(ctx context.Context, c jsmClient, name string) (err error) {
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
