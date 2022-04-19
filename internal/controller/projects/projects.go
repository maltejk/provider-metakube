/*
Copyright 2021 The Crossplane Authors.
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

package projects

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/google/go-cmp/cmp"

	"github.com/pkg/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	mk "github.com/maltejk/metakube-go-client/pkg/client"
	"github.com/maltejk/metakube-go-client/pkg/client/project"
	"github.com/maltejk/metakube-go-client/pkg/models"
	v1alpha1 "github.com/maltejk/provider-metakube/apis/projects/v1alpha1"

	mkClient "github.com/maltejk/provider-metakube/internal/client"
)

const (
	errNotProject     = "managed resource is not an Project custom resource"
	errCreateFailed   = "cannot create Project"
	errUpdateFailed   = "cannot update Project"
	errDescribeFailed = "cannot describe Project"
	errDeleteFailed   = "cannot delete Project"
)

// SetupProject adds a controller that reconciles Projects.
func SetupProject(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.ProjectKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.Project{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ProjectGroupVersionKind),
			managed.WithExternalConnecter(&connector{kube: mgr.GetClient(), newClientFn: mk.New}),
			managed.WithInitializers(),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

type connector struct {
	kube        client.Client
	newClientFn func(transport runtime.ClientTransport, formats strfmt.Registry) *mk.MetaKubeAPI
}

type external struct {
	client *mk.MetaKubeAPI
	kube   client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Project)
	if !ok {
		return nil, errors.New(errNotProject)
	}

	cfg, err := mkClient.GetConfig(ctx, c.kube, mg)
	if err != nil {
		return nil, err
	}

	client := c.newClientFn(cfg, strfmt.Default)
	return &external{client, c.kube}, nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProject)
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: false,
		}, nil
	}

	req := &project.GetProjectParams{
		ProjectID: id,
		Context:   ctx,
	}
	resp, reqErr := e.client.Project.GetProject(req, nil)
	if reqErr != nil {
		return managed.ExternalObservation{ResourceExists: false}, errors.Wrap(resource.Ignore(IsNotFound, reqErr), errDescribeFailed)
	}

	cr.Status.AtProvider = generateObservation(resp)

	currentSpec := cr.Spec.ForProvider.DeepCopy()

	cr.Status.SetConditions(v1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        isUpToDate(&cr.Spec.ForProvider, resp),
		ResourceLateInitialized: !cmp.Equal(&cr.Spec.ForProvider, currentSpec),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProject)
	}

	req := &project.CreateProjectParams{
		Body:       project.CreateProjectBody{Name: cr.Spec.ForProvider.Name, Labels: cr.Spec.ForProvider.Labels, Users: cr.Spec.ForProvider.Users},
		Context:    ctx,
		HTTPClient: &http.Client{},
	}
	resp, err := e.client.Project.CreateProject(req, nil)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateFailed)
	}

	meta.SetExternalName(cr, resp.Payload.ID)

	return managed.ExternalCreation{
		ExternalNameAssigned: true,
	}, nil

}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProject)
	}

	req := &project.UpdateProjectParams{
		Body: &models.Project{
			Annotations: cr.Spec.ForProvider.Annotations,
			ID:          cr.Status.AtProvider.ID,
			Labels:      cr.Spec.ForProvider.Labels,
			Name:        cr.Spec.ForProvider.Name,
		},
		ProjectID:  cr.Status.AtProvider.ID,
		Context:    ctx,
		HTTPClient: &http.Client{},
	}

	_, err := e.client.Project.UpdateProject(req, nil)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateFailed)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {

	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return errors.New(errNotProject)
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return errors.New(errNotProject)
	}

	req := &project.DeleteProjectParams{
		ProjectID:  id,
		Context:    ctx,
		HTTPClient: &http.Client{},
	}

	_, err := e.client.Project.DeleteProject(req, nil)
	if err != nil {
		return errors.Wrap(err, errDeleteFailed)
	}
	cr.Status.AtProvider = v1alpha1.ProjectObservation{}

	return nil
}

func generateObservation(in *project.GetProjectOK) v1alpha1.ProjectObservation {
	cr := v1alpha1.ProjectObservation{}

	obj := in.Payload
	cr.CreationTime = obj.CreationTimestamp.String()
	cr.ID = obj.ID
	cr.Name = obj.Name

	return cr
}

// isUpToDate checks whether there is a change in any of the modifiable fields.
func isUpToDate(cr *v1alpha1.ProjectParameters, gobj *project.GetProjectOK) bool { // nolint:gocyclo
	obj := gobj.Payload

	if !mkClient.IsEqualString(mkClient.StringToPtr(cr.Name), mkClient.StringToPtr(obj.Name)) {
		return false
	}

	if !cmp.Equal(cr.Labels, obj.Labels) {
		return false
	}
	return true
}
