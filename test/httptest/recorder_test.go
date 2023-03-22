// Copyright 2023 Harald Albrecht.
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

package httptest

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("wrapped HTTP response recorder", Ordered, func() {

	var msg string

	BeforeEach(func() {
		RegisterFailHandler(func(message string, callerSkip ...int) {
			msg = message
			RegisterFailHandler(Fail) // reset
		})
	})

	It("passes single use of WriteHeader", func() {
		rr := NewRecorder()
		rr.WriteHeader(200)
		Expect(msg).To(BeEmpty())
	})

	It("fails on superfluous WriteHeader call", func() {
		rr := NewRecorder()
		rr.WriteHeader(200)
		rr.WriteHeader(666)
		Expect(msg).To(ContainSubstring("superfluous response.WriteHeader call"))
	})

})
