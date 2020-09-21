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

package blockmap

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/govice/golinks/archivemap"

	"github.com/govice/golinks/fs"
	"github.com/govice/golinks/walker"
	"github.com/pkg/errors"

	"bytes"
	"crypto/sha512"

	"encoding/json"

	"fmt"

	"os"
)

//OutputName stores the default file name archive metadata
const OutputName string = ".link"

//BlockMap is a ad-hoc Merkle tree-map
type BlockMap struct {
	Archive     archivemap.ArchiveMap `json:"archive"`
	RootHash    []byte                `json:"rootHash"`
	Root        string                `json:"root"`
	IgnorePaths []string              `json:"ignorePaths"`
	AutoIgnore  bool                  `json:"autoIgnore"`
}

type IgnoredPathErr struct {
	Paths []string
}

func (ip *IgnoredPathErr) Error() string { return strings.Join(ip.Paths, " ,") }

//New returns a new BlockMap initialized at the provided root
func New(root string) *BlockMap {
	//Initialize map and assign blockmap root
	rootMap := make(archivemap.ArchiveMap)
	return &BlockMap{Archive: rootMap, RootHash: nil, Root: root, AutoIgnore: false}
}

//Generate creates an archive of the provided archives root filesystem
func (b *BlockMap) Generate() error {
	//Create a filesystem walker
	w := walker.New(b.Root)
	//Walk the root directory
	if err := w.Walk(); err != nil {
		return errors.Wrap(err, "BlockMap: failed to walk "+w.Root())
	}

	ignoredPath := func(ignoredPaths []string, value string) bool {
		for _, ip := range ignoredPaths {
			if strings.HasPrefix(value, ip) {
				return true
			}
		}
		return false
	}

	var ips *IgnoredPathErr
	//Iterate through all walked files
	for _, filePath := range w.Archive() {
		if ignoredPath(b.IgnorePaths, filePath) {
			continue
		}
		//Extract the relative path for the archive
		relPath, err := filepath.Rel(w.Root(), filePath)
		if err != nil {
			return errors.Wrap(err, "BlockMap: failed to extract relative file path")
		}

		//Ignore the files generated by this library
		if relPath == OutputName {
			continue
		}

		//Get the hash for the file
		fileHash, err := fs.HashFile(filePath)
		if err != nil {
			if err := errors.Unwrap(err); b.AutoIgnore && err != nil {
				if os.IsPermission(err) {
					b.AddIgnorePath(filePath)
					if ips == nil {
						ips = &IgnoredPathErr{
							Paths: []string{filePath},
						}
					} else {
						ips.Paths = append(ips.Paths, filePath)
					}
					continue
				}
			}
			return errors.Wrap(err, "BlockMap: failed to hash "+filePath)
		}

		//Use linux path seperator
		relPath = strings.Replace(relPath, "\\", "/", -1)

		//Add the hash to the archive using the relative path as it's key
		b.Archive[relPath] = fileHash
	}

	//If we're here, the entries are successful so we'll hash the blockmap.
	if err := b.hashBlockMap(); err != nil {
		return errors.Wrap(err, "blockmap: failed to generate block map")
	}

	if ips != nil && len(ips.Paths) > 0 {
		return ips
	}
	return nil
}

// SetIgnorePaths sets a list of paths to ignore in blockmap generation
func (b *BlockMap) SetIgnorePaths(paths []string) {
	b.IgnorePaths = uniqueStringSlice([]string{}, paths)
}

// AddIgnorePath adds a path to ignore during blockmap generation
func (b *BlockMap) AddIgnorePath(path string) {
	b.IgnorePaths = uniqueStringSlice(b.IgnorePaths, []string{path})
}

func uniqueStringSlice(original, additions []string) []string {
	unique := make(map[string]*struct{})
	for _, p := range original {
		unique[p] = &struct{}{}
	}

	for _, p := range additions {
		unique[p] = &struct{}{}
	}

	var out []string
	for p := range unique {
		out = append(out, p)
	}

	return out
}

func (b *BlockMap) hashBlockMap() error {
	if b.Archive == nil {
		return errors.New("blockmap: Attempted to hash null archive")
	}

	hash := sha512.New()
	archiveJSON, err := json.Marshal(b.Archive)
	if err != nil {
		return errors.Wrap(err, "blockmap: hash failed to encode archive map JSON")
	}
	if _, err := hash.Write(archiveJSON); err != nil {
		return errors.Wrap(err, "blockmap: failed to write to write hash buffer")
	}

	b.RootHash = hash.Sum(nil)
	return nil

}

//PrintBlockMap prints an existing block map and returns an error if not configured
func (b BlockMap) PrintBlockMap() {
	if b.RootHash == nil {
		log.Println("BlockMap is unhashed or unset")
	}
	fmt.Println("Root: " + b.Root)
	fmt.Printf("Hash: %v\n", b.RootHash)
	for key, value := range b.Archive {
		fmt.Printf("%v: %v\n", key, value)
	}
}

//Save will store a byte file of the blockmap in the default OutputFile
func (b BlockMap) Save(path string) error {
	return b.saveHelper(path, "")
}

//SaveNamed will store a byte file of the blockmap in the named OutputFile
func (b BlockMap) SaveNamed(path, name string) error {
	return b.saveHelper(path, name)
}

func (b BlockMap) saveHelper(path, name string) error {
	if b.RootHash == nil {
		return errors.New("BlockMap: can't save nil hashed map")
	}

	jsonBytes, err := json.Marshal(b)
	if err != nil {
		return errors.Wrap(err, "BlockMap: failed to encode link json")
	}
	linkFilePath := path + string(os.PathSeparator) + name + OutputName
	if err := ioutil.WriteFile(linkFilePath, jsonBytes, 0755); err != nil {
		return errors.Wrap(err, "BlockMap: failed to write to link")
	}

	return nil
}

//Load reads the blockmap from the default OutputFile
func (b *BlockMap) Load(path string) error {
	linkFilePath := path + string(os.PathSeparator) + OutputName
	jsonBytes, err := ioutil.ReadFile(linkFilePath)
	if err != nil {
		return errors.Wrap(err, "BlockMap: failed to read link file")
	}

	if err := json.Unmarshal(jsonBytes, &b); err != nil {
		return errors.Wrap(err, "BlockMap failed to unmarshal link json")
	}

	return nil
}

//Equal returns an evaluation of the equality of two blockmaps
func Equal(a, b *BlockMap) bool {
	if !bytes.Equal(a.RootHash, b.RootHash) {
		return false
	}

	aJSON, err := json.Marshal(a.Archive)
	if err != nil {
		panic(err)
	}

	bJSON, err := json.Marshal(b.Archive)
	if err != nil {
		panic(err)
	}

	return reflect.DeepEqual(aJSON, bJSON)
}
