package ginkgoextend

import "testing"

type Nested struct {
	Value string
}

type NestedNested struct {
	Nested Nested
}

type Complex struct {
	IntField     int
	StringField  string
	BoolField    bool
	PtrField     *int
	SliceInts    []int
	SliceStructs []Nested
	MapString    map[string]string
	MapStructs   map[string]Nested
	Nested       Nested
	DoubleNested NestedNested
}

func TestNonZeroMatcher(t *testing.T) {
	var ptrVal = 42

	actual := Complex{
		IntField:    10,
		StringField: "hello",
		BoolField:   true,
		PtrField:    &ptrVal,
		SliceInts:   []int{1, 2},
		SliceStructs: []Nested{
			{Value: "x"}, {Value: "y"},
		},
		MapString: map[string]string{"a": "1", "b": "2"},
		MapStructs: map[string]Nested{
			"a": {Value: "x"},
			"b": {Value: "y"},
		},
		Nested: Nested{Value: "nested"},
		DoubleNested: NestedNested{
			Nested: Nested{
				Value: "double-nested",
			},
		},
	}

	runTestSuccess := func(expected Complex) {
		success, err := MatchNonZero(expected).Match(actual)
		if !success || err != nil {
			t.Fatalf("Non-zero matcher failed: %v", err)
		}
	}
	runTestFail := func(expected Complex) {
		success, err := MatchNonZero(expected).Match(actual)
		if success || err == nil {
			t.Fatalf("Non-zero matcher did not fail: %v", err)
		}
	}

	t.Run("AllDefault", func(t *testing.T) {
		runTestSuccess(Complex{})
	})

	t.Run("IntFieldFail", func(t *testing.T) {
		runTestFail(Complex{
			IntField: 5,
		})
	})

	t.Run("IntFieldSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			IntField: 10,
		})
	})

	t.Run("PtrFieldFail", func(t *testing.T) {
		var i = 234242
		runTestFail(Complex{
			PtrField: &i,
		})
	})

	t.Run("PtrFieldSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			PtrField: &ptrVal,
		})
	})

	t.Run("StringFieldFail", func(t *testing.T) {
		runTestFail(Complex{
			StringField: "wrong",
		})
	})

	t.Run("StringFieldSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			StringField: "hello",
		})
	})

	t.Run("BoolFieldFail", func(t *testing.T) {
		runTestFail(Complex{
			BoolField: false,
		})
	})

	t.Run("BoolFieldSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			BoolField: true,
		})
	})

	t.Run("SliceIntsFail", func(t *testing.T) {
		runTestFail(Complex{
			SliceInts: []int{1, 3},
		})
	})

	t.Run("SliceIntsSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			SliceInts: []int{1, 2},
		})
	})

	t.Run("SliceStructsFail", func(t *testing.T) {
		runTestFail(Complex{
			SliceStructs: []Nested{{Value: "x"}, {Value: "wrong"}},
		})
	})

	t.Run("SliceStructsSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			SliceStructs: []Nested{{Value: "x"}, {Value: "y"}},
		})
	})

	t.Run("MapStringFail", func(t *testing.T) {
		runTestFail(Complex{
			MapString: map[string]string{"a": "wrong"},
		})
	})

	t.Run("MapStringSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			MapString: map[string]string{"a": "1"},
		})
	})

	t.Run("MapStructsFail", func(t *testing.T) {
		runTestFail(Complex{
			MapStructs: map[string]Nested{"a": {Value: "wrong"}},
		})
	})

	t.Run("MapStructsSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			MapStructs: map[string]Nested{"a": {Value: "x"}},
		})
	})

	t.Run("NestedFail", func(t *testing.T) {
		runTestFail(Complex{
			Nested: Nested{Value: "wrong"},
		})
	})

	t.Run("NestedSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			Nested: Nested{Value: "nested"},
		})
	})

	t.Run("DoubleNestedFail", func(t *testing.T) {
		runTestFail(Complex{
			DoubleNested: NestedNested{
				Nested: Nested{Value: "wrong"},
			},
		})
	})

	t.Run("DoubleNestedSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			DoubleNested: NestedNested{
				Nested: Nested{Value: "double-nested"},
			},
		})
	})

	t.Run("FullSuccess", func(t *testing.T) {
		runTestSuccess(Complex{
			IntField:  10,
			PtrField:  &ptrVal,
			SliceInts: []int{1, 2},
			SliceStructs: []Nested{
				{Value: "x"}, {Value: "y"},
			},
			MapString: map[string]string{"a": "1"},
			MapStructs: map[string]Nested{
				"a": {Value: "x"},
			},
			Nested: Nested{Value: "nested"},
			DoubleNested: NestedNested{
				Nested: Nested{
					Value: "double-nested",
				},
			},
		})
	})
}
