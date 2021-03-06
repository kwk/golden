package golden

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var updateGoldenFiles = flag.Bool("update", false, "when set, rewrite the golden files")

// CompareOptions define how the comparison and golden file generation will take
// place
type CompareOptions struct {
	// Whether or not to ignore UUIDs when comparing or writing the golden file
	// to disk. When this is on we replace UUIDs in both strings (the golden
	// file as well as in the actual object) before comparing the two strings.
	// This should make the comparison UUID agnostic without loosing the
	// locality comparison. In other words, that means we replace each UUID with
	// a more generic "00000000-0000-0000-0000-000000000001",
	// "00000000-0000-0000-0000-000000000002", ...,
	// "00000000-0000-0000-0000-00000000000N" value.
	UUIDAgnostic bool
	// Whether or not to ignore date/times when comparing or writing the golden
	// file to disk.  We replace all RFC3339 time strings with
	// "0001-01-01T00:00:00Z".
	DateTimeAgnostic bool
	// Whether or not to call JSON marshall on the actual object before
	// comparing it against the content of the golden file or writing to the
	// golden file. If this is false, then we will treat the actual object as a
	// []byte or string.
	MarshalInputAsJSON bool
}

// CompareWithGolden compares the actual object against the one from a
// golden file and let's you specify the options to be used for comparison and
// golden file production by hand. If the -update flag is given, that golden
// file is overwritten with the current actual object. When adding new tests you
// first must run them with the -update flag in order to create an initial
// golden version.
func CompareWithGolden(t *testing.T, goldenFile string, actualObj interface{}, opts CompareOptions) {
	if err := testableCompareWithGolden(*updateGoldenFiles, goldenFile, actualObj, opts); err != nil {
		t.Fatal(err)
	}
}

type stringer interface {
	String() string
}

func testableCompareWithGolden(update bool, goldenFile string, actualObj interface{}, opts CompareOptions) error {
	absPath, err := filepath.Abs(goldenFile)
	if err != nil {
		return fmt.Errorf("failed to get abosolute path for %q: %w", goldenFile, err)
	}
	var actual []byte
	if opts.MarshalInputAsJSON {
		var err error
		actual, err = json.MarshalIndent(actualObj, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal actual object: %w", err)
		}
	} else {
		switch t := actualObj.(type) {
		case []byte:
			actual = t
		case string:
			actual = []byte(t)
		case stringer:
			actual = []byte(t.String())
		default:
			return fmt.Errorf("don't know how to convert type of object %[1]T to string: %+[1]v (consider enabling MarshalInputAsJSON option): %w", actualObj, err)
		}
	}
	if update {
		// Make sure the directory exists where to write the file to
		err := os.MkdirAll(filepath.Dir(absPath), os.FileMode(0777))
		if err != nil {
			return fmt.Errorf("failed to create directory (and potential parents dirs) to write golden file to: %w", err)
		}

		tmp := string(actual)
		// Eliminate concrete UUIDs if requested. This makes adding changes to
		// golden files much more easy in git.
		if opts.UUIDAgnostic {
			tmp, err = replaceUUIDs(tmp)
			if err != nil {
				return fmt.Errorf("failed to replace UUIDs with more generic ones: %w", err)
			}
		}
		if opts.DateTimeAgnostic {
			tmp, err = replaceTimes(tmp)
			if err != nil {
				return fmt.Errorf("failed to replace RFC3339 times with default time: %w", err)
			}
		}
		err = ioutil.WriteFile(absPath, []byte(tmp), os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to update golden file %q: %w", absPath, err)
		}
	}
	expected, err := ioutil.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read golden file %q: %w", absPath, err)
	}

	expectedStr := string(expected)
	actualStr := string(actual)
	if opts.UUIDAgnostic {
		expectedStr, err = replaceUUIDs(expectedStr)
		if err != nil {
			return fmt.Errorf("failed to replace UUIDs with more generic ones: %w", err)
		}
		actualStr, err = replaceUUIDs(actualStr)
		if err != nil {
			return fmt.Errorf("failed to replace UUIDs with more generic ones: %w", err)
		}
	}
	if opts.DateTimeAgnostic {
		expectedStr, err = replaceTimes(expectedStr)
		if err != nil {
			return fmt.Errorf("failed to replace RFC3339 times with default time: %w", err)
		}
		actualStr, err = replaceTimes(actualStr)
		if err != nil {
			return fmt.Errorf("failed to replace RFC3339 times with default time: %w", err)
		}
	}
	if expectedStr != actualStr {
		// log.Printf("ERROR: testableCompareWithGolden: expected value %v", expectedStr)
		// log.Printf("ERROR: testableCompareWithGolden: actual value %v", actualStr)

		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expectedStr, actualStr, false)
		return fmt.Errorf("mismatch of actual output and golden-file %q:\n %s \n", absPath, dmp.DiffPrettyText(diffs))
	}
	return nil
}

// findUUIDs returns an array of uniq UUIDs that have been found in the given
// string
func findUUIDs(str string) ([]uuid.UUID, error) {
	pattern := "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}"
	uuidRegexp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile UUID regex pattern %q: %w", pattern, err)
	}
	uniqIDs := map[uuid.UUID]struct{}{}
	var res []uuid.UUID
	for _, uuidStr := range uuidRegexp.FindAllString(str, -1) {
		ID, err := uuid.FromString(uuidStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UUID %q: %w", uuidStr, err)
		}
		_, alreadyInMap := uniqIDs[ID]
		if !alreadyInMap {
			uniqIDs[ID] = struct{}{}
			// append to array
			res = append(res, ID)
		}
	}
	return res, nil
}

// replaceUUIDs finds all UUIDs in the given string and replaces them with
// "00000000-0000-0000-0000-000000000001,
// "00000000-0000-0000-0000-000000000002", ...,
// "00000000-0000-0000-0000-00000000000N"
func replaceUUIDs(str string) (string, error) {
	replacementPattern := "00000000-0000-0000-0000-%012d"
	ids, err := findUUIDs(str)
	if err != nil {
		return "", fmt.Errorf("failed to find UUIDs in string %q: %w", str, err)
	}
	newStr := str
	for idx, id := range ids {
		newStr = strings.Replace(newStr, id.String(), fmt.Sprintf(replacementPattern, idx+1), -1)
	}
	return newStr, nil
}

// replaceTimes finds all RFC3339 times and RFC7232 (section 2.2) times in the
// given string and replaces them with "0001-01-01T00:00:00Z" (for RFC3339) or
// "Mon, 01 Jan 0001 00:00:00 GMT" (for RFC7232) respectively.
func replaceTimes(str string) (string, error) {
	year := "([0-9]+)"
	month := "(0[1-9]|1[012])"
	day := "(0[1-9]|[12][0-9]|3[01])"
	datePattern := year + "-" + month + "-" + day

	hour := "([01][0-9]|2[0-3])"
	minute := "([0-5][0-9])"
	second := "([0-5][0-9]|60)"
	subSecond := "(\\.[0-9]+)?"
	timePattern := hour + ":" + minute + ":" + second + subSecond

	timeZoneOffset := "(([Zz])|([\\+|\\-]([01][0-9]|2[0-3]):[0-5][0-9]))"

	pattern := datePattern + "[Tt]" + timePattern + timeZoneOffset

	rfc3339Pattern, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile RFC3339 regex pattern %q: %w", pattern, err)
	}
	res := rfc3339Pattern.ReplaceAllString(str, `0001-01-01T00:00:00Z`)

	dayName := "(Mon|Tue|Wed|Thu|Fri|Sat|Sun)"
	day = "[0-9]{2}"
	month = "(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)"
	year = "[0-9]{4}"
	hour = "([01][0-9]|2[0-3])"
	minute = "([0-5][0-9])"
	second = "([0-5][0-9]|60)"
	tz := "(GMT|CEST|UTC|IST|[A-Z]+)"
	pattern = dayName + ", " + day + " " + month + " " + year + " " + hour + ":" + minute + ":" + second + " " + tz

	lastModifiedPattern, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile RFC7232 last-modified regex pattern %q: %w", pattern, err)
	}

	return lastModifiedPattern.ReplaceAllString(res, `Mon, 01 Jan 0001 00:00:00 GMT`), nil
}
