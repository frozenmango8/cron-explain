package cronexplain

import (
	"fmt"
	"testing"
)

func TestDescribe(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"* * * * *", "Every minute"},
		{"*/1 * * * *", "Every minute"},
		{"0 * * * *", "Every hour"},
		{"30 * * * *", "Every hour at minute 30"},
		{"15,45 * * * *", "Every hour at minutes 15 and 45"},
		{"*/15 * * * *", "Every 15 minutes"},
		{"1-5/2 * * * *", "Every hour at minutes 1, 3, and 5"},
		{"1-59/2 * * * *", "Every 2 minutes from minute 1 through 59"},
		{"0 0 * * *", "Every day at 00:00"},
		{"30 9 * * *", "Every day at 09:30"},
		{"0 9,17 * * *", "Every day at 09:00 and 17:00"},
		{"0 0,6,12,18 * * *", "Every day at 00:00, 06:00, 12:00, and 18:00"},
		{"*/15 9-17 * * *", "Every 15 minutes from 09:00 through 17:59"},
		{"* 9-17 * * *", "Every minute from 09:00 through 17:59"},
		{"0 9-17 * * *", "Every hour from 09:00 through 17:00"},
		{"30 9-17 * * *", "Every hour at minute 30 from 09:00 through 17:00"},
		{"*/15 9 * * *", "Every 15 minutes during hour 9"},
		{"*/15 9-17 * * 1-5", "Every 15 minutes from 09:00 through 17:59 on Monday through Friday"},
		{"0 9 * * MON", "At 09:00 on Monday"},
		{"0 9 * * MON,WED,FRI", "At 09:00 on Monday, Wednesday, and Friday"},
		{"0 9 * * 1-5", "At 09:00 on Monday through Friday"},
		{"0 0 * * 0", "At 00:00 on Sunday"},
		{"0 0 * * 7", "At 00:00 on Sunday"},
		{"0 0 * * FRI-SUN", "At 00:00 on Friday through Sunday"},
		{"0 0 * * SAT,SUN", "At 00:00 on Saturday through Sunday"},
		{"0 0 1 * *", "At 00:00 on day 1 of the month"},
		{"0 0 1,15 * *", "At 00:00 on days 1 and 15 of the month"},
		{"0 0 1-3 * *", "At 00:00 on days 1 through 3 of the month"},
		{"0 0 1 1 *", "At 00:00 on January 1"},
		{"0 0 * JAN *", "Every day at 00:00 in January"},
		{"0 0 * jan-mar *", "Every day at 00:00 in January through March"},
		{"0 0 * JAN,JUL *", "Every day at 00:00 in January and July"},
		{"*/15 * * JAN *", "Every 15 minutes in January"},
		{"0 12 * JAN MON", "At 12:00 on Monday in January"},
		{"30 4 1,15 * 5", "At 04:30 on days 1 and 15 of the month, or on Friday"},
		{"0 0 1 1 MON", "At 00:00 on January 1, or on Monday"},
		{"0 0 1 * 0-6", "Every day at 00:00"},
		{"@hourly", "Every hour"},
		{"@daily", "Every day at 00:00"},
		{"@midnight", "Every day at 00:00"},
		{"@weekly", "At 00:00 on Sunday"},
		{"@monthly", "At 00:00 on day 1 of the month"},
		{"@yearly", "At 00:00 on January 1"},
		{"@annually", "At 00:00 on January 1"},
		{"@reboot", "At startup"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			e, err := Parse(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			got := e.Describe()
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func ExampleExpr_Describe() {
	e, err := Parse("*/15 9-17 * * 1-5")
	if err != nil {
		panic(err)
	}
	fmt.Println(e.Describe())
	// Output:
	// Every 15 minutes from 09:00 through 17:59 on Monday through Friday
}
