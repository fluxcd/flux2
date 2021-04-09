/*
Copyright 2021 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sourcesecret

import (
	"io/ioutil"
	"reflect"
	"testing"
)

func Test_loadKeyPair(t *testing.T) {
	pk, _ := ioutil.ReadFile("testdata/rsa")
	ppk, _ := ioutil.ReadFile("testdata/rsa.pub")

	got, err := loadKeyPair("testdata/rsa")
	if err != nil {
		t.Errorf("loadKeyPair() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got.PrivateKey, pk) {
		t.Errorf("PrivateKey %s != %s", got.PrivateKey, pk)
	}
	if !reflect.DeepEqual(got.PublicKey, ppk) {
		t.Errorf("PublicKey %s != %s", got.PublicKey, ppk)
	}
}
