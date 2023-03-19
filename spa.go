// Copyright 2022 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spaserve

import (
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

// ForwardedPrefixHeader, if present, specifies the prefix that need to be
// preprended to the request's URI path in order to learn the original path
// when hitting the path rewriting proxy.
const ForwardedPrefixHeader = "X-Forwarded-Prefix"

// ForwardedUriHeader, if present, specifies the original URI (or sometimes only
// the original URI path) of a request when hitting the first path rewriting
// proxy.
const ForwardedUriHeader = "X-Forwarded-Uri"

// baseRe matches the base element in index.html in order to allow us to
// dynamically rewrite the base the SPA is served from. Please note that it
// doesn't make sense to use Go's templating here, as for development reasons
// the index.html must be perfectly usable without any Go templating at any
// time.
//
// Please note: "*?" instead of "*" ensures that our irregular expression
// doesn't get too greedy, gobbling much more than it should until the last(!)
// empty element.
var baseRe = regexp.MustCompile(`(<base href=").*?("\s*/>)`)

// SPAHandler implements an http.Handler that serves only the Index file on
// (almost) all request paths, except for static assets found in the
// StaticAssetsPath or any subdirectory thereof. The Index file contents served
// are automatically adjusted to the correct request base path, based on
// forwarding proxy headers.
type SPAHandler struct {
	fs                fs.FS         // the FS to serve static resources from.
	index             string        // (unrooted) path and name of the index/SPA file inside fs.
	staticfileHandler http.Handler  // FS adapted to http's file serving handler needs.
	indexRewriter     IndexRewriter // optional user function to rewrite the index/SPA file as necessary.
}

// NewSPAHandler returns a new HTTP handler serving static resources from the
// specified fs. It serves the index resource instead whenever no directly
// matching file can be found on the specified fs. The index resource should be
// specified as an unrooted, slash-separated path+name to be servable from the
// given fs; but NewSPAHandler will sanitize the index path anyway.
//
// The index parameter typically is "index.html"; please check with your SPA
// build environment documentation for the exact file name.
//
// In order to serve the static resources from a directory on the OS file
// system, use os.DirFS:
//
//	h := NewSPAHandler(os.DirFS("/opt/data/myspa"), "index.html")
func NewSPAHandler(fs fs.FS, index string, opts ...SPAHandlerOption) *SPAHandler {
	h := &SPAHandler{
		fs:                fs,
		staticfileHandler: http.FileServer(http.FS(fs)),
		index:             path.Clean("/" + index)[1:],
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// SPAHandlerOption sets optional properties at the time of creating an
// SPAHandler.
type SPAHandlerOption func(*SPAHandler)

// IndexRewriter rewrites (parts) of an index/SPA file contents to be delivered
// to a requesting client, after the base element has been updated. It can be
// optionally activated using the WithIndexRewriter option when creating a new
// SPAHandler.
type IndexRewriter func(r *http.Request, index string) string

// WithIndexRewriter sets the specified IndexRewriter that gets called before
// delivering the index/SPA file contents to requesting clients, allowing for
// application-specific changes.
func WithIndexRewriter(rewriter IndexRewriter) SPAHandlerOption {
	return func(h *SPAHandler) {
		h.indexRewriter = rewriter
	}
}

// ServeHTTP either serves a static resource when available inside
// SPAHandler.StaticAssetsPath or otherwise the specified Index asset inside the
// static assets everywhere else. This behavior is required for SPAs with
// client-side DOM routers, as otherwise bookmarking (router) links or reloading
// an SPA with the current route other than "/" would fail.
func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the absolute and also cleaned path to the requested resource in order
	// to prevent parent directory traversal outside the static assets
	// directory. Slapping "/" ensures that path.Clean does NOT to use the
	// current working dir for resolving the request path ... whichever current
	// working directory it might be at the moment is.
	r.URL.Path = path.Clean("/" + r.URL.Path)
	if h.serveStaticAsset(w, r) {
		return
	}
	h.serveRewrittenIndex(w, r)
}

// serveRewrittenIndex serves the index file, rewriting its HTML base element if
// found to refer the correct base path of the SPA.
func (h *SPAHandler) serveRewrittenIndex(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		if err != nil {
			NormalizedHttpError(w, err)
		}
	}()
	// Sanitize the base path so it cannot interfere with our regexp replacement
	// operations where we need to use "$1" and "$2" back references. As this
	// ain't VMS (shudder), we don't need "$" in SPA paths anyway.
	base := strings.ReplaceAll(h.basename(r), "$", "")
	// Grab the index.html's contents into a string as we need to modify it
	// on-the-fly based on where we deem the base path to be. And finally serve
	// the updated contents.
	f, err := h.fs.Open(h.index)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	fileInfo, err := f.Stat()
	if err != nil {
		return
	}
	indexhtmlcontents, err := io.ReadAll(f)
	if err != nil {
		return
	}
	finalIndexhtml := baseRe.ReplaceAllString(string(indexhtmlcontents), "${1}"+base+"${2}")
	if h.indexRewriter != nil {
		finalIndexhtml = h.indexRewriter(r, finalIndexhtml)
	}
	http.ServeContent(w, r, "index.html", fileInfo.ModTime(), strings.NewReader(finalIndexhtml))
}

// serveStaticAsset tries to serve a static asset specified in uripath from the
// SPAHandler's fs and returning true if successful. If no such static asset
// exists, nothing is served and false is returned instead.
//
// IMPORTANT: the passed r.URL.Path must have already been sanitized.
func (h *SPAHandler) serveStaticAsset(w http.ResponseWriter, r *http.Request) bool {
	// Try to check that the requested resource in fact is a plain file.
	// Thankfully, fs.State deals with fs.FS implementations that don't support
	// fs.StatFS and works around this situation. Thus, we can rely on fs.Stat
	// to give us stat information, if the file exists, whatever measures that
	// takes.
	path := r.URL.Path[1:] // ...fs.FS uses unrooted paths.
	if path == "" {
		return false // hitting root is always a case for index.html
	}
	info, err := fs.Stat(h.fs, r.URL.Path[1:])
	// If we have a "regular" file then serve it using a regular
	// http.FileServer. Fun fact: http.FileServer also sanitizes our already
	// sanitized path.
	if err == nil && info.Mode()&os.ModeType == 0 {
		h.staticfileHandler.ServeHTTP(w, r)
		return true
	}
	// If we got an error and it isn't a missing static asset, then normalize
	// (or rather, sanitize) the error and send that back to the client.
	if err != nil && !os.IsNotExist(err) {
		NormalizedHttpError(w, err)
		return true
	}
	return false
}

// originalReqPath returns the (hopefully) original path when hitting the first
// proxy in a chain, based on what has been passed down to us. If no suitable
// forwarding information is present, the original -- and already sanitized --
// request URL path.
func (h *SPAHandler) originalReqPath(r *http.Request) string {
	// Was the request path rewritten? Then the original request path was the
	// forwarded prefix, followed by the remaining part we now see in the
	// request.
	if fwprefix := r.Header.Get(ForwardedPrefixHeader); fwprefix != "" {
		fwprefix = path.Clean("/" + fwprefix)
		return path.Join(fwprefix, r.URL.Path)
	}
	// Was the original HTTP request URL passed upon us? There seem to be
	// different interpretations with some proxy implementations only passing
	// the request path, but not the full original URI to us...
	if fwurl := r.Header.Get(ForwardedUriHeader); fwurl != "" {
		if strings.HasPrefix(fwurl, "/") {
			// Assume it's just the request path: sani, sani, sanitize it!
			return path.Clean(fwurl)
		}
		// Attempt to parse it as a URI, erm, URL, and sani, sani, sanitize it!;
		// if that fails, just ignore it.
		if u, err := url.Parse(fwurl); err == nil {
			return path.Clean("/" + u.Path)
		}
	}
	// If nothing else, go with just the request path we see.
	return r.URL.Path
}

// basename returns the URI request path base based on the given request, by
// consulting proxy headers when available. Rewriting forwarding proxies need to
// preserve the original client-side request URI path for this to work; if
// deriving the base name is impossible, the base is taken to be "/" from the
// clients' perspective.
func (h *SPAHandler) basename(r *http.Request) string {
	reqPath := r.URL.Path
	originalReqPath := h.originalReqPath(r)
	var base string
	if strings.HasSuffix(reqPath, "/") && !strings.HasSuffix(originalReqPath, "/") {
		// take care of the situation where the reverse proxy redirects from
		// /foo to /foo/ and then rewrites the path to /.
		originalReqPath += "/"
	}
	// If the request path we see is a proper suffix of the original request
	// path, take only the common base part (~prefix).
	if strings.HasSuffix(originalReqPath, reqPath) {
		base = originalReqPath[:len(originalReqPath)-len(reqPath)]
	}
	// Ensure that the base path always ends with a "/", as otherwise
	// browsers will throw the specified path under the bus (erm, nevermind)
	// of a dirname() operation, clipping off the final element that once
	// was a proper directory name. Oh, well.
	if strings.HasSuffix(base, "/") {
		return base
	}
	return base + "/"
}
