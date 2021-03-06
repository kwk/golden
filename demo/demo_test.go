package demo_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kwk/golden"
	uuid "github.com/satori/go.uuid"
)

type Address struct {
	Street   string
	Number   int
	Postcode string
	Country  string
}
type Person struct {
	FirstName string
	LastName  string
	Address   Address
}

func TestDemo1(t *testing.T) {
	johnDoe := Person{
		FirstName: "John",
		LastName:  "Doe",
		Address: Address{
			Street:  "Avenue Lane",
			Number:  3,
			Country: "North Pole",
		},
	}

	golden.CompareWithGolden(t, "johnDoe.golden.json", johnDoe, golden.CompareOptions{MarshalInputAsJSON: true})
}

func TestAddMovedInField(t *testing.T) {
	// Let's augment the Person struct by
	type PersonMovedIn struct {
		Person
		MovedIn time.Time
	}

	johnDoe := PersonMovedIn{
		Person: Person{
			FirstName: "John",
			LastName:  "Doe",
			Address: Address{
				Street:  "Avenue Lane",
				Number:  3,
				Country: "North Pole",
			},
		},
		MovedIn: time.Now(),
	}

	golden.CompareWithGolden(t, "movedIn.golden.json", johnDoe, golden.CompareOptions{
		MarshalInputAsJSON: true,
		DateTimeAgnostic:   true,
	})
}

func TestIgnoredField(t *testing.T) {
	type IgnoredField struct {
		A string
		b string
	}

	actual := IgnoredField{
		A: "Hello",
		b: "world",
	}

	golden.CompareWithGolden(t, "ignoredField.golden.json", actual, golden.CompareOptions{MarshalInputAsJSON: true})
}

type StructWithPrivateField struct {
	A string
	b string
}

func (s StructWithPrivateField) String() string {
	return fmt.Sprintf("A=%q\nb=%q", s.A, s.b)
}

func TestPrivateFieldButIncludedInString(t *testing.T) {
	actual := StructWithPrivateField{
		A: "Hello",
		b: "world",
	}

	golden.CompareWithGolden(t, "structWithPrivateField.golden.json", actual, golden.CompareOptions{MarshalInputAsJSON: false})
}

func TestSillyUUIDStruct(t *testing.T) {
	// Let's augment the Person struct by
	type UUIDGroup struct {
		A, B, C, D, E, F uuid.UUID
	}

	x := uuid.NewV4()
	y := uuid.NewV4()
	z := uuid.NewV4()

	actual := UUIDGroup{y, z, x, z, x, y}

	golden.CompareWithGolden(t, "sillyUuid.golden.json", actual, golden.CompareOptions{
		MarshalInputAsJSON: true,
		UUIDAgnostic:       true,
	})
}
