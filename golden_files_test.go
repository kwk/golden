package golden

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

const testInputStr = `
{
	"data": {
		"attributes": {
		"createdAt": "2017-04-21T04:38:26.777609Z",
		"last_used_workspace": "my-last-used-workspace",
		"type": "git",
		"url": "https://github.com/fabric8-services/fabric8-wit.git"
		},
		"id": "d7a282f6-1c10-459e-bb44-55a1a6d48bdd",
		"links": {
		"edit": "http:///api/codebases/d7a282f6-1c10-459e-bb44-55a1a6d48bdd/edit",
		"related": "http:///api/codebases/d7a282f6-1c10-459e-bb44-55a1a6d48bdd",
		"self": "http:///api/codebases/d7a282f6-1c10-459e-bb44-55a1a6d48bdd"
		},
		"relationships": {
		"space": {
			"data": {
			"id": "a8bee527-12d2-4aff-9823-3511c1c8e6b9",
			"type": "spaces"
			},
			"links": {
			"related": "http:///api/spaces/a8bee527-12d2-4aff-9823-3511c1c8e6b9",
			"self": "http:///api/spaces/a8bee527-12d2-4aff-9823-3511c1c8e6b9"
			}
		}
		},
		"type": "codebases"
	}
}`

const testUUIDOutputStr = `
{
	"data": {
		"attributes": {
		"createdAt": "2017-04-21T04:38:26.777609Z",
		"last_used_workspace": "my-last-used-workspace",
		"type": "git",
		"url": "https://github.com/fabric8-services/fabric8-wit.git"
		},
		"id": "00000000-0000-0000-0000-000000000001",
		"links": {
		"edit": "http:///api/codebases/00000000-0000-0000-0000-000000000001/edit",
		"related": "http:///api/codebases/00000000-0000-0000-0000-000000000001",
		"self": "http:///api/codebases/00000000-0000-0000-0000-000000000001"
		},
		"relationships": {
		"space": {
			"data": {
			"id": "00000000-0000-0000-0000-000000000002",
			"type": "spaces"
			},
			"links": {
			"related": "http:///api/spaces/00000000-0000-0000-0000-000000000002",
			"self": "http:///api/spaces/00000000-0000-0000-0000-000000000002"
			}
		}
		},
		"type": "codebases"
	}
}`

func TestGoldenFindUUIDs(t *testing.T) {
	t.Parallel()
	t.Run("find UUIDs", func(t *testing.T) {
		t.Parallel()
		ids, err := findUUIDs(testInputStr)
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{
			uuid.FromStringOrNil("d7a282f6-1c10-459e-bb44-55a1a6d48bdd"),
			uuid.FromStringOrNil("a8bee527-12d2-4aff-9823-3511c1c8e6b9"),
		}, ids)
	})
}

func TestGoldenReplaceUUIDs(t *testing.T) {
	t.Parallel()
	t.Run("replace UUIDs", func(t *testing.T) {
		t.Parallel()
		newStr, err := replaceUUIDs(testInputStr)
		require.NoError(t, err)
		require.Equal(t, testUUIDOutputStr, newStr)
	})
}

const testTimesOutputStr = `
{
	"data": {
		"attributes": {
		"createdAt": "0001-01-01T00:00:00Z",
		"last_used_workspace": "my-last-used-workspace",
		"type": "git",
		"url": "https://github.com/fabric8-services/fabric8-wit.git"
		},
		"id": "d7a282f6-1c10-459e-bb44-55a1a6d48bdd",
		"links": {
		"edit": "http:///api/codebases/d7a282f6-1c10-459e-bb44-55a1a6d48bdd/edit",
		"related": "http:///api/codebases/d7a282f6-1c10-459e-bb44-55a1a6d48bdd",
		"self": "http:///api/codebases/d7a282f6-1c10-459e-bb44-55a1a6d48bdd"
		},
		"relationships": {
		"space": {
			"data": {
			"id": "a8bee527-12d2-4aff-9823-3511c1c8e6b9",
			"type": "spaces"
			},
			"links": {
			"related": "http:///api/spaces/a8bee527-12d2-4aff-9823-3511c1c8e6b9",
			"self": "http:///api/spaces/a8bee527-12d2-4aff-9823-3511c1c8e6b9"
			}
		}
		},
		"type": "codebases"
	}
}`

func TestGoldenReplaceTimes(t *testing.T) {
	t.Parallel()
	t.Run("rfc3339", func(t *testing.T) {
		t.Parallel()
		newStr, err := replaceTimes(testInputStr)
		require.NoError(t, err)
		require.Equal(t, testTimesOutputStr, newStr)
	})
	timeStrings := map[string]string{
		"rfc7232":                  `"last-modified": "Thu, 15 Mar 2018 09:23:37 GMT",`,
		"arbitrary date":           `"last-modified": "Fri, 13 Apr 2018 16:21:50 CEST",`,
		"date with IST timezone":   `"last-modified": "Mon, 23 Apr 2018 00:00:00 IST",`,
		"Bangladesh Standard Time": `"last-modified": "Mon, 24 Apr 2018 02:11:00 BST",`,
	}
	for timeType, timeString := range timeStrings {
		t.Run(timeType, func(t *testing.T) {
			t.Parallel()
			expected := `"last-modified": "Mon, 01 Jan 0001 00:00:00 GMT",`
			actual, err := replaceTimes(timeString)
			// then
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}
}

func TestGoldenCompareWithGolden(t *testing.T) {
	t.Parallel()
	type Foo struct {
		ID        uuid.UUID
		Bar       string
		CreatedAt time.Time
	}
	dummy := Foo{Bar: "hello world", ID: uuid.NewV4()}
	dummyStr := uuid.NewV4().String()

	agnosticOpts := []CompareOptions{
		{UUIDAgnostic: true, DateTimeAgnostic: true, MarshalInputAsJSON: true},
		{UUIDAgnostic: true, DateTimeAgnostic: true, MarshalInputAsJSON: false},
		{UUIDAgnostic: true, DateTimeAgnostic: false, MarshalInputAsJSON: true},
		{UUIDAgnostic: true, DateTimeAgnostic: false, MarshalInputAsJSON: false},
		{UUIDAgnostic: false, DateTimeAgnostic: true, MarshalInputAsJSON: true},
		{UUIDAgnostic: false, DateTimeAgnostic: true, MarshalInputAsJSON: false},
		{UUIDAgnostic: false, DateTimeAgnostic: false, MarshalInputAsJSON: true},
		{UUIDAgnostic: false, DateTimeAgnostic: false, MarshalInputAsJSON: false},
	}
	for _, opts := range agnosticOpts {
		t.Run("file not found", func(t *testing.T) {
			// given
			f := "not_existing_file.golden.json"
			// when
			var data interface{} = dummy
			if !opts.MarshalInputAsJSON {
				data = dummyStr
			}
			err := testableCompare(false, f, data, opts)
			// then
			require.Error(t, err)
			var pathError *os.PathError
			require.True(t, errors.As(err, &pathError))
		})
		t.Run("update golden file in a folder that does not yet exist", func(t *testing.T) {
			// given
			f := "not/existing/folder/file.golden.json"
			// when
			var data interface{} = dummy
			if !opts.MarshalInputAsJSON {
				data = dummyStr
			}
			err := testableCompare(true, f, data, opts)
			// then
			// then double check that file exists and no error occurred
			require.NoError(t, err)
			_, err = os.Stat(f)
			require.NoError(t, err)
			require.NoError(t, os.Remove(f), "failed to remove test file")
			require.NoError(t, os.Remove("not/existing/folder"))
			require.NoError(t, os.Remove("not/existing"))
			require.NoError(t, os.Remove("not/"))
		})
		t.Run("mismatch between expected and actual output", func(t *testing.T) {
			// given
			f, err := ioutil.TempFile(".", "")
			require.NoError(t, err)
			defer os.Remove(f.Name())
			// when
			var data interface{} = dummy
			if !opts.MarshalInputAsJSON {
				data = dummyStr
			}
			err = testableCompare(false, f.Name(), data, opts)
			// then
			require.Error(t, err)
			var pathError *os.PathError
			require.False(t, errors.As(err, &pathError))
		})
	}

	t.Run("comparing with existing file", func(t *testing.T) {
		// given
		tempFile, err := ioutil.TempFile(".", "")
		require.NoError(t, err)
		f := tempFile.Name()
		bs, err := json.MarshalIndent(dummy, "", "  ")
		require.NoError(t, err)
		err = ioutil.WriteFile(f, bs, os.ModePerm)
		require.NoError(t, err)
		defer func() {
			err := os.Remove(f)
			require.NoError(t, err)
		}()

		t.Run("comparing with the same object", func(t *testing.T) {
			t.Run("not agnostic", func(t *testing.T) {
				// when
				err = testableCompare(false, f, dummy, CompareOptions{MarshalInputAsJSON: true})
				// then
				require.NoError(t, err)
			})
			t.Run("agnostic", func(t *testing.T) {
				// when
				err = testableCompare(false, f, dummy, CompareOptions{UUIDAgnostic: true, DateTimeAgnostic: true, MarshalInputAsJSON: true})
				// then
				require.NoError(t, err)
			})
		})
		t.Run("comparing with the same object but modified its UUID", func(t *testing.T) {
			dummy.ID = uuid.NewV4()
			t.Run("not agnostic", func(t *testing.T) {
				// when
				err = testableCompare(false, f, dummy, CompareOptions{MarshalInputAsJSON: true})
				// then
				require.Error(t, err)
			})
			t.Run("agnostic", func(t *testing.T) {
				// when
				err = testableCompare(false, f, dummy, CompareOptions{UUIDAgnostic: true, DateTimeAgnostic: true, MarshalInputAsJSON: true})
				// then
				require.NoError(t, err)
			})
		})
		t.Run("comparing with the same object but modified its time", func(t *testing.T) {
			dummy.CreatedAt = time.Now()
			t.Run("not agnostic", func(t *testing.T) {
				// when
				err = testableCompare(false, f, dummy, CompareOptions{MarshalInputAsJSON: true})
				// then
				require.Error(t, err)
			})
			t.Run("agnostic", func(t *testing.T) {
				// when
				err = testableCompare(false, f, dummy, CompareOptions{UUIDAgnostic: true, DateTimeAgnostic: true, MarshalInputAsJSON: true})
				// then
				require.NoError(t, err)
			})
		})
	})
}
