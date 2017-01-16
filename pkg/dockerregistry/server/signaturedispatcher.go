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
	Name    string `json:"name"`
	Type    string `json:"type"`
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

// SignatureDispatcher dispatch the signatures endpoint.
func SignatureDispatcher(ctx *handlers.Context, r *http.Request) http.Handler {
	signatureHandler := &signatureHandler{
		Context: ctx,
	}
	signatureHandler.Reference, _ = imageapi.ParseDockerImageReference(ctxu.GetStringValue(ctx, "vars.name") + "@" + ctxu.GetStringValue(ctx, "vars.digest"))
	return gorillahandlers.MethodHandler{
		"GET": http.HandlerFunc(signatureHandler.Get),
		"PUT": http.HandlerFunc(signatureHandler.Put),
	}
}

type signatureHandler struct {
	Context   *handlers.Context
	Reference imageapi.DockerImageReference
}

func (s *signatureHandler) Put(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(s.Context).Debugf("(*signatureHandler).Put")
	client, ok := UserClientFrom(s.Context)
	if !ok {
		s.handleError(s.Context, errcode.ErrorCodeUnknown.WithDetail("unable to get origin client"), w)
		return
	}

	sig := signature{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.handleError(s.Context, ErrorCodeSignatureInvalid.WithDetail(err.Error()), w)
		return
	}
	context.GetLogger(s.Context).Debugf("(*signatureHandler).Put: %s", string(body))
	if err := json.Unmarshal(body, &sig); err != nil {
		s.handleError(s.Context, ErrorCodeSignatureInvalid.WithDetail(err.Error()), w)
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
			s.handleError(s.Context, errcode.ErrorCodeUnauthorized.WithDetail(err.Error()), w)
		case kapierrors.IsBadRequest(err):
			s.handleError(s.Context, ErrorCodeSignatureInvalid.WithDetail(err.Error()), w)
		case kapierrors.IsNotFound(err):
			w.WriteHeader(http.StatusNotFound)
		case kapierrors.IsAlreadyExists(err):
			s.handleError(s.Context, ErrorCodeSignatureAlreadyExists.WithDetail(err.Error()), w)
		default:
			s.handleError(s.Context, errcode.ErrorCodeUnknown.WithDetail(fmt.Sprintf("unable to create image %q signature: %v", s.Reference.String(), err)), w)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *signatureHandler) Get(w http.ResponseWriter, req *http.Request) {
	context.GetLogger(s.Context).Debugf("(*signatureHandler).Get")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	client, ok := UserClientFrom(s.Context)
	if !ok {
		s.handleError(s.Context, errcode.ErrorCodeUnknown.WithDetail("unable to get origin client"), w)
		return
	}

	if len(s.Reference.Namespace) == 0 || len(s.Reference.Name) == 0 || len(s.Reference.ID) == 0 {
		s.handleError(s.Context, v2.ErrorCodeNameInvalid.WithDetail("invalid image format"), w)
		return
	}

	image, err := client.ImageStreamImages(s.Reference.Namespace).Get(s.Reference.Name, s.Reference.ID)
	if err != nil {
		switch {
		case kapierrors.IsUnauthorized(err):
			s.handleError(s.Context, errcode.ErrorCodeUnauthorized.WithDetail(fmt.Sprintf("not authorized to get image %q signature: %v", s.Reference.String(), err)), w)
		case kapierrors.IsNotFound(err):
			w.WriteHeader(http.StatusNotFound)
		default:
			s.handleError(s.Context, errcode.ErrorCodeUnknown.WithDetail(fmt.Sprintf("unable to get image %q signature: %v", s.Reference.String(), err)), w)
		}
		return
	}

	signatures := signatureList{Signatures: []signature{}}
	for _, s := range image.Image.Signatures {
		signatures.Signatures = append(signatures.Signatures, signature{
			Name:    s.Name,
			Type:    s.Type,
			Content: s.Content,
		})
	}

	if data, err := json.Marshal(signatures); err != nil {
		s.handleError(s.Context, errcode.ErrorCodeUnknown.WithDetail(fmt.Sprintf("failed to serialize image signature %v", err)), w)
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
