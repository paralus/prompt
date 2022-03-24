package mock

import (
	"net/http"

	authv3 "github.com/RafayLabs/rcloud-base/pkg/auth/v3"
	commonv3 "github.com/RafayLabs/rcloud-base/proto/types/commonpb/v3"
	"github.com/urfave/negroni"
)

type authMiddleware struct{}

func NewDummyAuthMiddleware() negroni.Handler {
	return &authMiddleware{}
}

func (am *authMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	sd := &commonv3.SessionData{
		Account:       "dummy",
		Organization:  "dummy",
		Partner:       "dummy",
		Role:          "dummy",
		Permissions:   []string{"dummy"},
		PartnerDomain: "dummy",
		Username:      "dummy",
		Groups:        []string{"dummy"},
	}
	ctx := authv3.NewSessionContext(r.Context(), sd)
	next(rw, r.WithContext(ctx))
}
