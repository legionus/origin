package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	kapierrors "k8s.io/kubernetes/pkg/api/errors"

	ctxu "github.com/docker/distribution/context"

	"github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/registry/handlers"

	imageapi "github.com/openshift/origin/pkg/image/api"

	gorillahandlers "github.com/gorilla/handlers"
)

type signature struct {
	// Name must be in "sha256:<digest>@signatureName" format
	Name string `json:"name"`
	// Type is optional, of not set it will be defaulted to "AtomicImageV1"
	Type string `json:"type"`
	// Content contains the base64 encoded GPG signature
	Content []byte `json:"content"`
}

type signatureList struct {
	Signatures []signature `json:"signatures"`
}

const errGroup = "registry.api.v2"

var (
	ErrorCodeSignatureInvalid = errcode.Register(errGroup, errcode.ErrorDescriptor{
		Value:          "SIGNATURE_INVALID",
		Message:        "invalid image signature",
		HTTPStatusCode: http.StatusBadRequest,
	})

	ErrorCodeSignatureAlreadyExists = errcode.Register(errGroup, errcode.ErrorDescriptor{
		Value:          "SIGNATURE_EXISTS",
		Message:        "image signature already exists",
		HTTPStatusCode: http.StatusBadRequest,
	})
)

type signatureHandler struct {
	ctx       *handlers.Context
	reference imageapi.DockerImageReference
}

// SignatureDispatcher handles the GET and PUT requests for signature endpoint.
func SignatureDispatcher(ctx *handlers.Context, r *http.Request) http.Handler {
	signatureHandler := &signatureHandler{ctx: ctx}
	signatureHandler.reference, _ = imageapi.ParseDockerImageReference(ctxu.GetStringValue(ctx, "vars.name") + "@" + ctxu.GetStringValue(ctx, "vars.digest"))
	return gorillahandlers.MethodHandler{
		"GET": http.HandlerFunc(signatureHandler.Get),
		"PUT": http.HandlerFunc(signatureHandler.Put),
	}
}

func (s *signatureHandler) Put(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(s.ctx).Debugf("(*signatureHandler).Put")
	client, ok := UserClientFrom(s.ctx)
	if !ok {
		s.handleError(s.ctx, errcode.ErrorCodeUnknown.WithDetail("unable to get origin client"), w)
		return
	}

	sig := signature{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.handleError(s.ctx, ErrorCodeSignatureInvalid.WithDetail(err.Error()), w)
		return
	}
	if err := json.Unmarshal(body, &sig); err != nil {
		s.handleError(s.ctx, ErrorCodeSignatureInvalid.WithDetail(err.Error()), w)
		return
	}

	if len(sig.Type) == 0 {
		sig.Type = imageapi.ImageSignatureTypeAtomicImageV1
	}
	newSig := &imageapi.ImageSignature{Content: sig.Content, Type: sig.Type}
	newSig.Name = sig.Name

	if _, err := client.ImageSignatures().Create(newSig); err != nil {
		switch {
		case kapierrors.IsUnauthorized(err):
			s.handleError(s.ctx, errcode.ErrorCodeUnauthorized.WithDetail(err.Error()), w)
		case kapierrors.IsBadRequest(err):
			s.handleError(s.ctx, ErrorCodeSignatureInvalid.WithDetail(err.Error()), w)
		case kapierrors.IsNotFound(err):
			w.WriteHeader(http.StatusNotFound)
		case kapierrors.IsAlreadyExists(err):
			s.handleError(s.ctx, ErrorCodeSignatureAlreadyExists.WithDetail(err.Error()), w)
		default:
			s.handleError(s.ctx, errcode.ErrorCodeUnknown.WithDetail(fmt.Sprintf("unable to create image %q signature: %v", s.reference.String(), err)), w)
		}
		return
	}

	// Return just 201 with no body.
	// TODO: The docker registry actually returns the Location header
	w.WriteHeader(http.StatusCreated)
}

func (s *signatureHandler) Get(w http.ResponseWriter, req *http.Request) {
	context.GetLogger(s.ctx).Debugf("(*signatureHandler).Get")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	client, ok := UserClientFrom(s.ctx)
	if !ok {
		s.handleError(s.ctx, errcode.ErrorCodeUnknown.WithDetail("unable to get origin client"), w)
		return
	}

	if len(s.reference.ID) == 0 {
		s.handleError(s.ctx, v2.ErrorCodeNameInvalid.WithDetail("the image ID must be specified (sha256:<digest>"), w)
		return
	}

	image, err := client.ImageStreamImages(s.reference.Namespace).Get(s.reference.Name, s.reference.ID)
	if err != nil {
		switch {
		case kapierrors.IsUnauthorized(err):
			s.handleError(s.ctx, errcode.ErrorCodeUnauthorized.WithDetail(fmt.Sprintf("not authorized to get image %q signature: %v", s.reference.String(), err)), w)
		case kapierrors.IsNotFound(err):
			w.WriteHeader(http.StatusNotFound)
		default:
			s.handleError(s.ctx, errcode.ErrorCodeUnknown.WithDetail(fmt.Sprintf("unable to get image %q signature: %v", s.reference.String(), err)), w)
		}
		return
	}

	// Transform the OpenShift ImageSignature into Registry signature object.
	signatures := signatureList{Signatures: []signature{}}
	for _, s := range image.Image.Signatures {
		signatures.Signatures = append(signatures.Signatures, signature{
			Name:    s.Name,
			Type:    s.Type,
			Content: s.Content,
		})
	}

	if data, err := json.Marshal(signatures); err != nil {
		s.handleError(s.ctx, errcode.ErrorCodeUnknown.WithDetail(fmt.Sprintf("failed to serialize image signature %v", err)), w)
	} else {
		w.Write(data)
	}
}

func (s *signatureHandler) handleError(ctx context.Context, err error, w http.ResponseWriter) {
	context.GetLogger(ctx).Errorf("(*signatureHandler): %v", err)
	ctx, w = context.WithResponseWriter(ctx, w)
	if serveErr := errcode.ServeJSON(w, err); serveErr != nil {
		context.GetResponseLogger(ctx).Errorf("error sending error response: %v", serveErr)
		return
	}
}
