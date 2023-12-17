package server

import (
	"net/http"
	"net/url"

	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/gin-gonic/gin"
)

type strHandler struct {
	str storage.Storer
}

func (o *strHandler) Redirect(c *gin.Context) {
	lookup := &url.URL{
		Host: c.Request.Host,
		Path: c.Request.URL.Path,
	}

	red, err := o.str.Get(lookup)
	if err == storage.ErrNotFound {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if err == nil {
		c.Status(http.StatusTemporaryRedirect)
		c.Header("Location", red.String())
		return
	}

	// The error returned here is not actionable within this function; error handling is deferred to middleware.
	_ = c.AbortWithError(http.StatusInternalServerError, err)
}
