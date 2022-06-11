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
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("normalize errors into HTTP status codes", func() {

	DescribeTable("normalize errors",
		func(err error, expected int) {
			w := httptest.NewRecorder()
			NormalizedHttpError(w, err)
			Expect(w.Result().StatusCode).To(Equal(expected))
		},
		Entry("something's missing", fmt.Errorf("foobar mistake %w", fs.ErrNotExist),
			http.StatusNotFound),
		Entry("something's out of reach", fmt.Errorf("finger wech! %w", fs.ErrPermission),
			http.StatusForbidden),
		Entry("else it's a server error", errors.New("foobar"),
			http.StatusInternalServerError),
	)

})
