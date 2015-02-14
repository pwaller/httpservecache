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

// Default handler keys on the r.URL.String().
func DefaultRequestKey(r *http.Request) string {
	return r.URL.String()
}

type requestContext struct {
	cacheHandler
	*http.Request
}

// A Group wraps a groupcache.Group.
// It provides two methods, `H` and `F` which wrap a `http.Handler` and
// `http.HandlerFunc` respectively.
// The wrapped handlers only invoke the underlying handler if the request has
// not yet been cached. The cache key is determined by the `requestKey`
// function. The underlying handler is invoked on whatever `*http.Request`
// happens to need the cache filling, so the http.Handler's response MUST only
// depend on variables of the `*http.Request` which the `requestKey` uses to
// construct the key.
type Group struct {
	*groupcache.Group
	requestKey requestKey

	// If nil, shows an ugly message.
	// TODO(pwaller): communicate the original `err` via the *r.Request, maybe
	// through some sort of context library?
	errorHandler http.Handler
}

// Construct a new `Group` called `groupName`.
// `requestKey` is used to determine how requests are cached. Two requests both
// which both product the same value for `requestKey(r)` will be considered
// equivalent in the eyes of the cache.
// `sizeMiB` specifies how much memory the groupcache is allowed to use.
// If `requestKey` is nil, the r.URL.String() is used.
func New(
	groupName string,
	requestKey func(r *http.Request) string,
	sizeMiB int64,
) *Group {
	size := sizeMiB << 20

	if requestKey == nil {
		requestKey = DefaultRequestKey
	}

	group := &Group{
		requestKey:   requestKey,
		errorHandler: nil, // TODO(pwaller): provide access
	}

	getter := groupcache.GetterFunc(group.fill)
	group.Group = groupcache.NewGroup(groupName, size, getter)
	return group
}

// Perform the http request on the underlying http Handler.
func (c *Group) fill(
	ctx groupcache.Context,
	key string,
	dest groupcache.Sink,
) error {
	rc, ok := ctx.(requestContext)
	if !ok {
		log.Panicf("Group.Get: expected RequestContext, got %T", ctx)
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

// Wrap a http.Handler.
func (c *Group) H(h http.Handler) http.Handler {
	return cacheHandler{c, h}
}

// Wrap a http.HandlerFunc.
func (c *Group) F(h http.HandlerFunc) http.Handler {
	return cacheHandler{c, h}
}

// http.Handler which wraps a particular http.Handler
type cacheHandler struct {
	*Group
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
