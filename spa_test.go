// Copyright 2022 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy
// of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package spaserve

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"

	"github.com/PuerkitoBio/goquery"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

//go:embed test/*
var embeddedFiles embed.FS
var embStaticFs, _ = fs.Sub(embeddedFiles, "test")

var _ = Describe("", func() {

	DescribeTable("test has embedded files correctly set up",
		func(name string) {
			f := Successful(embStaticFs.Open(name))
			f.Close()
		},
		Entry("index.html", "index.html"),
		Entry("static/js/some.js", "static/js/some.js"),
	)

	DescribeTable("determines original request path",
		func(path string, header http.Header, expected string) {
			url := Successful(url.Parse("http://foo.bar:12345" + path))
			r := &http.Request{
				Method: "GET",
				URL:    url,
				Header: header,
			}
			h := NewSPAHandler(embStaticFs, "index.html")
			Expect(h.originalReqPath(r)).To(Equal(expected))
		},
		Entry("/ without proxy headers", "/", nil, "/"),

		Entry("a request path without proxy headers", "/some/path", nil, "/some/path"),
		Entry("/ with X-Forwarded-Prefix header", "/", http.Header{
			ForwardedPrefixHeader: []string{"/"},
		}, "/"),
		Entry("/ with X-Forwarded-Prefix header", "/", http.Header{
			ForwardedPrefixHeader: []string{"/prefix"},
		}, "/prefix"),
		Entry("/foo with X-Forwarded-Prefix header", "/foo", http.Header{
			ForwardedPrefixHeader: []string{"/prefix"},
		}, "/prefix/foo"),

		Entry("/ with X-Forwarded-Uri path-only header", "/", http.Header{
			ForwardedUriHeader: []string{"/"},
		}, "/"),
		Entry("/ with X-Forwarded-Uri path-only empty header", "/", http.Header{
			ForwardedUriHeader: []string{""},
		}, "/"),
		Entry("/ with X-Forwarded-Uri path-only /prefix header", "/", http.Header{
			ForwardedUriHeader: []string{"/prefix"},
		}, "/prefix"),
		Entry("/ with X-Forwarded-Uri schemed header", "/", http.Header{
			ForwardedUriHeader: []string{"http://foo.bar:12345/prefix"},
		}, "/prefix"),
		Entry("/ with X-Forwarded-Uri schemed header", "/", http.Header{
			ForwardedUriHeader: []string{"http://foo.bar:12345/prefix/"},
		}, "/prefix"),
	)

	DescribeTable("determines basename path",
		func(path string, header http.Header, expected string) {
			url := Successful(url.Parse("http://foo.bar:12345" + path))
			r := &http.Request{
				Method: "GET",
				URL:    url,
				Header: header,
			}
			h := NewSPAHandler(embStaticFs, "index.html")
			Expect(h.basename(r)).To(Equal(expected))
		},
		Entry("/ without proxy headers", "/", nil, "/"),
		Entry("/foo/bar without proxy headers", "/foo/bar", nil, "/"),

		Entry("/ rewritten with prefix /foo", "/", http.Header{
			ForwardedPrefixHeader: []string{"/foo"},
		}, "/foo/"),
		Entry("/foo/bar rewritten with prefix /", "/foo/bar", http.Header{
			ForwardedPrefixHeader: []string{"/"},
		}, "/"),
		Entry("/foo/bar rewritten with empty prefix", "/foo/bar", http.Header{
			ForwardedPrefixHeader: []string{""},
		}, "/"),
		Entry("/foo/bar rewritten with prefix /foo", "/foo/bar", http.Header{
			ForwardedPrefixHeader: []string{"/foo"},
		}, "/foo/"),
		Entry("/foo/bar rewritten with prefix /foo/", "/foo/bar", http.Header{
			ForwardedPrefixHeader: []string{"/foo/"},
		}, "/foo/"),
		Entry("/foo/bar rewritten with prefix /bar", "/foo/bar", http.Header{
			ForwardedPrefixHeader: []string{"/bar"},
		}, "/bar/"), // sic!
		Entry("/foo/bar rewritten with prefix /foo/bar/", "/foo/bar", http.Header{
			ForwardedPrefixHeader: []string{"/foo/bar/"},
		}, "/foo/bar/"), // request outside, so clamp to prefix
	)

	DescribeTable("serves static content with correct status code",
		func(path, prefix string, expectedServed bool, expectedCanary string, expectedStatus int) {
			url := Successful(url.Parse("http://foo.bar:12345" + path))
			r := &http.Request{
				Method: "GET",
				URL:    url,
				Header: http.Header{
					ForwardedPrefixHeader: []string{prefix},
				},
			}
			h := NewSPAHandler(embStaticFs, "index.html")
			w := httptest.NewRecorder()
			Expect(h.serveStaticAsset(w, r)).To(Equal(expectedServed))
			switch expectedStatus {
			case 0:
			case http.StatusOK:
				contents := Successful(w.Body.ReadBytes('\n'))
				Expect(string(contents)).To(ContainSubstring(expectedCanary))
			default:
				Expect(w.Result().StatusCode).To(Equal(expectedStatus))
			}
		},
		Entry("/static/js/some.js",
			"/static/js/some.js", "/", true, "CANARY JS", http.StatusOK),
		Entry("[/foo]/static/js/some.js",
			"/static/js/some.js", "/foo", true, "CANARY JS", http.StatusOK),
		Entry("/static/foo/bar",
			"/static/foo/bar", "/", false, "", 0),
		Entry("/static",
			"/static", "/", false, "", 0),
		Entry("/static",
			"/..", "/..", true, "", 0),
	)

	DescribeTable("rewrites the index file",
		func(path, prefix string, expected string) {
			url := Successful(url.Parse("http://foo.bar:12345" + path))
			r := &http.Request{
				Method: "GET",
				URL:    url,
				Header: http.Header{
					ForwardedPrefixHeader: []string{prefix},
				},
			}
			h := NewSPAHandler(embStaticFs, "index.html")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			Expect(w.Result().StatusCode).To(Equal(http.StatusOK))
			doc, err := goquery.NewDocumentFromReader(w.Body)
			Expect(err).NotTo(HaveOccurred())
			base := doc.Find("base")
			Expect(base.Length()).To(Equal(1), "<base> element lost")
			href, _ := base.First().Attr("href")
			Expect(href).To(Equal(expected))
		},
		Entry("prefix /foo", "/bar/baz", "/foo", "/foo/"),
		Entry("/", "/", "/", "/"),
	)

	It("supports application-specific rewriting/post-processing", func() {
		url := Successful(url.Parse("http://foo.bar:12345"))
		r := &http.Request{
			Method: "GET",
			URL:    url,
		}
		const canary = "<!-- SOMETHING DIFFERENT -->"
		h := NewSPAHandler(embStaticFs, "index.html",
			WithIndexRewriter(func(r *http.Request, index string) string {
				return index + canary
			}))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		Expect(w.Result().StatusCode).To(Equal(http.StatusOK))
		Expect(w.Body).To(HaveSuffix(canary))
	})

	It("returns a 500 when the index is missing", func() {
		url := Successful(url.Parse("http://foo.bar:12345"))
		r := &http.Request{
			Method: "GET",
			URL:    url,
		}
		h := NewSPAHandler(embStaticFs, "bonkers.html")
		w := httptest.NewRecorder()
		h.serveRewrittenIndex(w, r)
		Expect(w.Result().StatusCode).To(Equal(http.StatusNotFound))
	})

	DescribeTable("serves a static asset using varying fs.FS implementations",
		func(fs fs.FS) {
			url := Successful(url.Parse("http://foo.bar:12345/icon.png"))
			r := &http.Request{
				Method: "GET",
				URL:    url,
			}
			h := NewSPAHandler(fs, "index.html")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			Expect(w.Result().StatusCode).To(Equal(http.StatusOK))
		},
		Entry("from embedded fs", embStaticFs),
		Entry("from test dir fs", os.DirFS("./test")),
	)

})
