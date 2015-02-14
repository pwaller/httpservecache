// Work in progress. HTTP middleware which automatically groupcaches things.
//
package httpservecache

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/golang/groupcache"

	"github.com/pwaller/httpservecache/pb"
)

// Function used to determine the key.
type requestKey func(r *http.Request) string

func (f requestKey) requestKey(r *http.Request) string { return f(r) }

// Default handler keys on the r.URL.Path.
func DefaultRequestKey(r *http.Request) string {
	return r.URL.Path
}

type requestContext struct {
	cacheHandler
	*http.Request
}

//
type Cacher struct {
	*groupcache.Group
	requestKey requestKey

	// If nil, shows an ugly message.
	// TODO(pwaller): communicate the original `err` via the *r.Request, maybe
	// through some sort of context library?
	errorHandler http.Handler
}

// Construct a new Cacher
func New(groupName string, rq func(r *http.Request) string, sizeMiB int64) *Cacher {
	size := sizeMiB << 20

	if rq == nil {
		rq = DefaultRequestKey
	}

	cacher := &Cacher{
		requestKey:   rq,
		errorHandler: nil, // TODO(pwaller): provide access
	}

	getter := groupcache.GetterFunc(cacher.fill)
	cacher.Group = groupcache.NewGroup(groupName, size, getter)
	return cacher
}

// Perform the http request on the underlying http Handler.
func (c *Cacher) fill(
	ctx groupcache.Context,
	key string,
	dest groupcache.Sink,
) error {
	rc, ok := ctx.(requestContext)
	if !ok {
		log.Panicf("Cacher.Get: expected RequestContext, got %T", ctx)
	}

	w := httptest.NewRecorder()
	rc.Handler.ServeHTTP(w, rc.Request)

	var response pb.Response
	response.Body = w.Body.Bytes()
	response.Code = new(int32)
	*response.Code = int32(w.Code)

	for h, values := range w.Header() {
		for _, value := range values {
			header := new(pb.Header)
			header.Key, header.Value = &h, &value
			response.Headers = append(response.Headers, header)
		}
	}

	err := dest.SetProto(&response)
	if err != nil {
		return err
	}
	return nil

}

// Wrap a http.Handler
func (c *Cacher) H(h http.Handler) http.Handler {
	return cacheHandler{c, h}
}

// Wrap a http.HandlerFunc
func (c *Cacher) F(h http.HandlerFunc) http.Handler {
	return cacheHandler{c, h}
}

// http.Handler which wraps a particular http.Handler
type cacheHandler struct {
	*Cacher
	http.Handler
}

// Fetch the response via groupcache
func (ch cacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var response pb.Response

	ctx := requestContext{ch, r}

	err := ch.Get(ctx, ch.requestKey(r), groupcache.ProtoSink(&response))
	if err != nil {
		if ch.errorHandler != nil {
			ch.errorHandler.ServeHTTP(w, r)
			return
		}
		msg := fmt.Sprintf("Cache failed: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	hs := w.Header()

	for _, h := range response.GetHeaders() {
		k := h.GetKey()
		hs[k] = append(hs[k], h.GetValue())
	}

	w.WriteHeader(int(response.GetCode()))

	_, err = w.Write(response.GetBody())
	if err != nil {
		log.Printf("Response write failed: %v", err)
	}
}
