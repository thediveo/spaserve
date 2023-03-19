// Copyright 2023 Harald Albrecht.
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

/*
Package httptest wraps the standard library's httptest.ResponseRecorder in order
to fail any test doing superfluous response.WriteHeader calls.
*/
package httptest

import (
	stdhttptest "net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// WrappedResponseRecorder wraps httptest.ResponseRecorder in order to fail
// tests doing superfluous WriteHeader calls.
type WrappedResponseRecorder struct {
	*stdhttptest.ResponseRecorder
	wroteHeader bool
}

// NewRecorder returns a new test response recorder detecting superfluous
// WriteHeader calls.
func NewRecorder() *WrappedResponseRecorder {
	return &WrappedResponseRecorder{
		ResponseRecorder: stdhttptest.NewRecorder(),
	}
}

// WriteHeader implements http.ResponseWriter, failing tests that do superfluous
// WriteHeader calls.
func (w *WrappedResponseRecorder) WriteHeader(code int) {
	GinkgoHelper()
	Expect(w.wroteHeader).To(BeFalse(), "superfluous response.WriteHeader call")
	w.wroteHeader = true
	w.ResponseRecorder.WriteHeader(code)
}
