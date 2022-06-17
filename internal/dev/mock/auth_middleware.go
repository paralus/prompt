package mock

import (
	"context"
	"net/http"

	"github.com/paralus/paralus/pkg/common"
	commonv3 "github.com/paralus/paralus/proto/types/commonpb/v3"
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

	ctx := context.WithValue(r.Context(), common.SessionDataKey, sd)
	next(rw, r.WithContext(ctx))
}
