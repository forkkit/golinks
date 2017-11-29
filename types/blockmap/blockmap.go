package blockmap

import (
	"log"
	"path/filepath"

	"bytes"
	"crypto/sha512"

	"encoding/gob"

	"fmt"

	"os"

	"github.com/LaughingCabbage/goLinks/types/fs"
	"github.com/LaughingCabbage/goLinks/types/walker"
	"github.com/pkg/errors"
)

//OutputName stores the default file name archive metadata
const OutputName string = ".link"

//BlockMap is a ad-hoc Merkle tree-map
type BlockMap struct {
	Archive  map[string][]byte
	RootHash []byte
	Root     string
}

//New returns a new BlockMap initialized at the provided root
func New(root string) *BlockMap {
	//Initialize map and assign blockmap root
	rootMap := make(map[string][]byte)
	return &BlockMap{Archive: rootMap, RootHash: nil, Root: root}
}

//Generate creates an archive of the provided archives root filesystem
func (b *BlockMap) Generate() error {
	//Create a filesystem walker
	w := walker.New(b.Root)
	//Walk the root directory
	if err := w.Walk(); err != nil {
		return errors.Wrap(err, "BlockMap: failed to walk "+w.Root())
	}

	//Iterate through all walked files
	for _, filePath := range w.Archive() {
		//Get the hash for the file
		fileHash, err := fs.HashFile(filePath)
		if err != nil {
			return errors.Wrap(err, "BlockMap: failed to hash "+filePath)
		}

		//Extract the relative path for the archive
		relPath, err := filepath.Rel(w.Root(), filePath)
		if err != nil {
			return errors.Wrap(err, "BlockMap: failed to extract relative file path")
		}

		//Add the hash to the archive using the relative path as it's key
		b.Archive[relPath] = fileHash
	}

	//If we're here, the entries are successful so we'll hash the blockmap.
	if err := b.hashBlockMap(); err != nil {
		return errors.Wrap(err, "BlockMap: failed to has block map")
	}

	return nil

}

func (b *BlockMap) hashBlockMap() error {
	//Make sure an archive exists
	if b.Archive == nil {
		return errors.New("hashBlockMap: Attempted to hash null archive")
	}

	//Begin hashing blockmap gob
	hash := sha512.New()
	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(b.Archive)
	if err != nil {
		return errors.Wrap(err, "hashBlockMap: failed to encode archive map")
	}
	if _, err := hash.Write(buffer.Bytes()); err != nil {
		return errors.Wrap(err, "hashBlockMap: failed to write to write hash buffer")
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
	if b.RootHash == nil {
		return errors.New("BlockMap: can't save nil hashed map")
	}
	file, err := os.OpenFile(path+string(os.PathSeparator)+OutputName, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return errors.Wrap(err, "BlockMap: failed to save link file")
	}
	encoder := gob.NewEncoder(file)
	if err = encoder.Encode(b); err != nil {
		return errors.Wrap(err, "BlockMap: failed to encode link file")
	}
	if err := file.Close(); err != nil {
		return errors.Wrap(err, "BlockMap: failed to close link file")
	}
	return nil
}

//Load reads the blockmap from the default OutputFile
func (b *BlockMap) Load(path string) error {
	file, err := os.Open(path + string(os.PathSeparator) + OutputName)
	if err != nil {
		return errors.Wrap(err, "BlockMap: failed to open link file")
	}
	decoder := gob.NewDecoder(file)
	if err = decoder.Decode(b); err != nil {
		return errors.Wrap(err, "BlockMap: failed to decode blockmap")
	}
	if err = file.Close(); err != nil {
		return errors.Wrap(err, "BlockMap: failed to close link file")
	}

	return nil
}

//Equal returns an evaluation of the equality of two blockmaps
func Equal(a, b *BlockMap) bool {
	if a.Root != b.Root && !bytes.Equal(a.RootHash, b.RootHash) {
		return false
	}
	return true
}