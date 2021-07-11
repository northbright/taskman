package filehash

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding"
	"encoding/json"
	"errors"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"strconv"

	"github.com/northbright/taskman"
)

type Task struct {
	File       string   `json:"file"`
	HashFuncs  []string `json:"hash_funcs"`
	BufferSize string   `json:"buffer_size"`
	hashes     map[string]hash.Hash
	bufferSize int
	total      int64
	summed     int64
	buf        []byte
	f          *os.File
	r          *bufio.Reader
}

type State struct {
	Summed string            `json:"summed"`
	Datas  map[string][]byte `json:"datas"`
	summed int64
}

type Result struct {
	Checksums map[string][]byte `json:"checksums"`
}

var (
	DefaultBufferSize int = 8 * 1024 * 1024

	NotSupportedHashFuncErr = errors.New("not supported hash func")
	NoFileToHashErr         = errors.New("no file to hash")
	HashFuncNotAvailableErr = errors.New("hash function is not available")
	InvalidHashStateErr     = errors.New("invalid hash state")
	FileIsDirErr            = errors.New("file is dir")
	SavedStateNotMatchedErr = errors.New("saved state and hash func not matched")
)

func init() {
	taskman.Register("filehash", loadTask)
}

func loadTask(data []byte) (taskman.Task, error) {
	var err error
	t := &Task{}
	t.hashes = make(map[string]hash.Hash)

	if err = json.Unmarshal(data, &t); err != nil {
		return nil, err
	}

	supportedHashFuncs := map[string]hash.Hash{
		"crc32":  crc32.NewIEEE(),
		"md5":    md5.New(),
		"sha1":   sha1.New(),
		"sha256": sha256.New(),
	}

	// Init hashes.
	for _, h := range t.HashFuncs {
		if _, ok := supportedHashFuncs[h]; !ok {
			return nil, NotSupportedHashFuncErr
		}
		t.hashes[h] = supportedHashFuncs[h]
	}

	// Get buffer size.
	if t.bufferSize, err = strconv.Atoi(t.BufferSize); err != nil {
		t.bufferSize = DefaultBufferSize
	}

	if t.bufferSize <= 0 {
		t.bufferSize = DefaultBufferSize
	}

	return t, nil
}

func (t *Task) MarshalBinary() ([]byte, error) {
	state := t.newState()
	return json.Marshal(state)
}

func (t *Task) UnmarshalBinary(data []byte) error {
	var err error

	state := State{}

	if err = json.Unmarshal(data, &state); err != nil {
		return err
	}

	if state.summed, err = strconv.ParseInt(state.Summed, 10, 64); err != nil {
		return err
	}

	// Restore saved hash states
	for k, v := range t.hashes {
		u := v.(encoding.BinaryUnmarshaler)

		data, ok := state.Datas[k]
		if !ok {
			return SavedStateNotMatchedErr
		}

		if err := u.UnmarshalBinary(data); err != nil {
			return err
		}
	}

	// Seek file offset and update summed size.
	if _, err := t.f.Seek(state.summed, os.SEEK_SET); err != nil {
		return err
	}

	// Update summed size and progress.
	t.summed = state.summed

	return nil
}

func (t *Task) Init(ctx context.Context) error {
	// Open file.
	if err := t.openFile(); err != nil {
		return err
	}

	// Create bufio.Reader.
	t.r = bufio.NewReaderSize(t.f, t.bufferSize)
	t.buf = make([]byte, t.bufferSize)

	t.summed = int64(0)

	return nil
}

func (t *Task) Deinit(ctx context.Context) error {
	if t.f != nil {
		t.f.Close()
	}
	return nil
}

func (t *Task) Step() (int64, bool, error) {
	n, err := t.r.Read(t.buf)
	if err != nil && err != io.EOF {
		return t.summed, true, err
	}

	if n == 0 {
		return t.summed, true, nil
	} else {

		// Adds more data to the running hash.
		for _, h := range t.hashes {
			if n, err = h.Write(t.buf[:n]); err != nil {
				return t.summed, true, err
			}
		}

		t.summed += int64(n)

		return t.summed, false, nil
	}
}

func (t *Task) Result() ([]byte, error) {
	result := Result{}
	result.Checksums = make(map[string][]byte)

	for k, v := range t.hashes {
		result.Checksums[k] = v.Sum(nil)
	}

	return json.Marshal(result)
}

func (t *Task) Total() int64 {
	return t.total
}

func (t *Task) openFile() error {
	f, err := os.Open(t.File)
	if err != nil {
		return err
	}

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return FileIsDirErr
	}

	t.f = f
	t.total = fi.Size()

	return nil
}

// newState returns the current hash state.
func (t *Task) newState() *State {
	datas := map[string][]byte{}

	for k, v := range t.hashes {
		m := v.(encoding.BinaryMarshaler)
		data, _ := m.MarshalBinary()
		datas[k] = data
	}

	state := &State{Summed: strconv.FormatInt(t.summed, 10), Datas: datas}
	return state
}
