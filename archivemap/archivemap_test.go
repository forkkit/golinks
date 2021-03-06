/*
 *Copyright 2018-2019 Kevin Gentile
 *
 *Licensed under the Apache License, Version 2.0 (the "License");
 *you may not use this file except in compliance with the License.
 *You may obtain a copy of the License at
 *
 *http://www.apache.org/licenses/LICENSE-2.0
 *
 *Unless required by applicable law or agreed to in writing, software
 *distributed under the License is distributed on an "AS IS" BASIS,
 *WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *See the License for the specific language governing permissions and
 *limitations under the License.
 */

package archivemap

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"
)

var goldenArchiveJSON = "{\"a1\":\"Ik5EZz0i\",\"a2\":\"Ik5Eaz0i\",\"a3\":\"Ik5UQT0i\"}"

func Test_UnmarshalJSON(t *testing.T) {
	am := ArchiveMap{}
	if err := am.UnmarshalJSON([]byte(goldenArchiveJSON)); err != nil {
		t.Error(err)
	}

	a1, _ := base64.StdEncoding.DecodeString("Ik5EZz0i")
	a2, _ := base64.StdEncoding.DecodeString("Ik5Eaz0i")
	a3, _ := base64.StdEncoding.DecodeString("Ik5UQT0i")
	if !bytes.Equal(am["a1"], a1) || !bytes.Equal(am["a2"], a2) || !bytes.Equal(am["a3"], a3) {
		t.Log(am)
		t.Error("Marshal Ordering failed " + goldenArchiveJSON)
	}

}

var goldenArchiveJSON2 = `{"C:/User/folder1":"Ik5EZz0i","C:/User/folder2":"Ik5EZz0i"}`

func Test_MarshalJSON(t *testing.T) {
	ps := string(os.PathSeparator)
	a1, _ := base64.StdEncoding.DecodeString("Ik5EZz0i")
	for i := 0; i < 10; i++ {
		am := ArchiveMap{
			"C:" + ps + "User" + ps + "folder1": a1,
			"C:" + ps + "User" + ps + "folder2": a1,
		}
		archivemapJSON, err := am.MarshalJSON()
		if err != nil {
			t.Error(err)
		}

		if !bytes.Equal(archivemapJSON, []byte(goldenArchiveJSON2)) {
			t.Error("Marshal Ordering failed " + string(archivemapJSON) + " | " + goldenArchiveJSON2)
		}
	}
}
