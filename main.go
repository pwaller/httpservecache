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
type RequestKey func(r *http.Request) string

func (f RequestKey) RequestKey(r *http.Request) string { return f(r) }

func DefaultRequestKey(r *http.Request) string {
	return r.URL.Path
}

type RequestContext struct {
	CacheHandler
	*http.Request
}

//
type Cacher struct {
	bucket, keyPrefix string
	*groupcache.Group
	requestKey RequestKey

	// If nil, shows an ugly message.
	// TODO(pwaller): communicate the original `err` via the *r.Request, maybe
	// through some sort of context library?
	errorHandler http.Handler
}

// Construct a new Cacher
func NewCacher(bucket, prefix string, rq RequestKey, sizeMiB int64) *Cacher {

	groupName := fmt.Sprint("servecache://", bucket, "/", prefix)
	size := sizeMiB << 20

	if rq == nil {
		rq = DefaultRequestKey
	}

	cacher := &Cacher{
		bucket:       bucket,
		keyPrefix:    prefix,
		requestKey:   rq,
		errorHandler: nil, // TODO(pwaller): provide access
	}

	cacher.Group = groupcache.NewGroup(groupName, size, cacher)
	return cacher
}

// Perform the http request on the underlying http Handler.
func (c *Cacher) Get(
	ctx groupcache.Context,
	key string,
	dest groupcache.Sink,
) error {
	rc, ok := ctx.(RequestContext)
	if !ok {
		return fmt.Errorf("Cacher.Get: expected RequestContext, got %T", ctx)
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
	return CacheHandler{c, h}
}

// Wrap a http.HandlerFunc
func (c *Cacher) F(h http.HandlerFunc) http.Handler {
	return CacheHandler{c, h}
}

// http.Handler which wraps a particular http.Handler
type CacheHandler struct {
	*Cacher
	http.Handler
}

// Fetch the response via groupcache
func (ch CacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var response pb.Response

	ctx := RequestContext{ch, r}

	err := ch.Group.Get(ctx, ch.requestKey(r), groupcache.ProtoSink(&response))
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
