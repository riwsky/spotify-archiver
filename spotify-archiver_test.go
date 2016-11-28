package spotify_archiver

import "testing"
import "time"

func TestCloneName(t *testing.T) {
	ts := time.Date(2016, time.October, 6, 0, 0, 0, 0, time.UTC)
	expected := "omg 10/3/2016"
	input := "omg"
	actual := cloneName(input, ts)
	if actual != expected {
		t.Errorf("cloneName(%v, %v) == %v != %v", input, ts, actual, expected)
	}
}

func TestGetMonday(t *testing.T) {

	makeOct := func(day int) time.Time {
		return time.Date(2016, time.October, day, 0, 0, 0, 0, time.UTC)
	}
	cases := []struct {
		input          time.Time
		expectedMonday time.Time
	}{
		{makeOct(6), makeOct(3)},
		{makeOct(3), makeOct(3)},
		{makeOct(1), time.Date(2016, time.September, 26, 0, 0, 0, 0, time.UTC)},
		{time.Date(2016, time.September, 26, 5, 3, 2, 200, time.UTC), time.Date(2016, time.September, 26, 0, 0, 0, 0, time.UTC)},
	}
	for _, testCase := range cases {
		actual := lessThanEqualMonday(testCase.input)
		if actual != testCase.expectedMonday {
			t.Errorf("lessThanEqualMonday(%v) == %v != %v", testCase.input, actual, testCase.expectedMonday)
		}
	}
}
