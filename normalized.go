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
	"errors"
	"io/fs"
	"net/http"
)

// NormalizedHttpError writes a normalized HTTP error message and HTTP status
// code based on the specified error, but not leaking any interesting internal
// server details from this specified error.
func NormalizedHttpError(w http.ResponseWriter, err error) {
	if errors.Is(err, fs.ErrNotExist) {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, fs.ErrPermission) {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}
	http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
}
