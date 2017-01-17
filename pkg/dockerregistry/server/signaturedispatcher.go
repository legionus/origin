package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api"
	kapierrors "k8s.io/kubernetes/pkg/api/errors"

	ctxu "github.com/docker/distribution/context"

	"github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/docker/distribution/registry/handlers"

	imageapi "github.com/openshift/origin/pkg/image/api"
	imageapiv1 "github.com/openshift/origin/pkg/image/api/v1"

	gorillahandlers "github.com/gorilla/handlers"
)

type ImageSignatureList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []imageapiv1.ImageSignature `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// SignatureDispatcher dispatch the signatures endpoint.
func SignatureDispatcher(ctx *handlers.Context, r *http.Request) http.Handler {
	signatureHandler := &signatureHandler{Context: ctx, originalReference: ctxu.GetStringValue(ctx, "vars.reference")}
	signatureHandler.Reference, _ = imageapi.ParseDockerImageReference(signatureHandler.originalReference)
	return gorillahandlers.MethodHandler{
		"GET": http.HandlerFunc(signatureHandler.Get),
	}
}

type signatureHandler struct {
	Context           *handlers.Context
	Reference         imageapi.DockerImageReference
	originalReference string
}

// Get serves the /signatures/<reference> endpoint. It requires the user token for the
// OpenShift authorization to fetch the ImageStreamImage and extract the Docker image
// signature from it which is then returned as versioned array to client.
func (s *signatureHandler) Get(w http.ResponseWriter, req *http.Request) {
	context.GetLogger(s.Context).Debugf("(*signatureHandler).Get")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	client, ok := UserClientFrom(s.Context)
	if !ok {
		s.handleError(s.Context, errcode.ErrorCodeUnknown.WithDetail("unable to get origin client"), w)
		return
	}

	if len(s.Reference.Namespace) == 0 || len(s.Reference.Name) == 0 || len(s.Reference.ID) == 0 {
		s.handleError(s.Context, errcode.ErrorCodeUnsupported.WithDetail(fmt.Sprintf("invalid image format %v", s.originalReference)), w)
		return
	}

	image, err := client.ImageStreamImages(s.Reference.Namespace).Get(s.Reference.Name, s.Reference.ID)
	if err != nil {
		if kapierrors.IsUnauthorized(err) {
			s.handleError(s.Context, errcode.ErrorCodeUnauthorized.WithDetail(fmt.Sprintf("not authorized to get image %q signature: %v", s.Reference.String(), err)), w)
			return
		}
		if kapierrors.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		s.handleError(s.Context, errcode.ErrorCodeUnknown.WithDetail(fmt.Sprintf("unable to get image %q signature: %v", s.Reference.String(), err)), w)
		return
	}

	result := &ImageSignatureList{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "ImageSignatureList",
			APIVersion: "registry.openshift.io/v1",
		},
		Items: []imageapiv1.ImageSignature{},
	}

	if len(image.Image.Signatures) == 0 {
		data, _ := json.Marshal(result)
		w.Write(data)
		return
	}

	if err := api.Scheme.Convert(&image.Image.Signatures, &result.Items, nil); err != nil {
		s.handleError(s.Context, errcode.ErrorCodeUnknown.WithDetail(fmt.Sprintf("failed to convert image signature to versioned object %v", err)), w)
		return
	}

	data, err := json.Marshal(result)
	if err != nil {
		s.handleError(s.Context, errcode.ErrorCodeUnknown.WithDetail(fmt.Sprintf("failed to serialize image signature %v", err)), w)
		return
	}

	w.Write(data)
}

func (s *signatureHandler) handleError(ctx context.Context, err error, w http.ResponseWriter) {
	context.GetLogger(s.Context).Errorf("(*signatureHandler).Get: %v", err)
	ctx, w = context.WithResponseWriter(ctx, w)
	if serveErr := errcode.ServeJSON(w, err); serveErr != nil {
		context.GetResponseLogger(ctx).Errorf("error sending error response: %v", serveErr)
		return
	}
}
