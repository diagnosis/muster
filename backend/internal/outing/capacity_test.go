// backend/internal/outing/capacity_test.go
// Tests for capacityOf, the one home of the seat and spot math.
package outing

import (
	"testing"
)

func Test_CapacityOf(t *testing.T) {
	tests := []struct {
		name                                 string
		maxSize, hostSeats                   int
		acceptedPeople, driverSeats, waiting int
		want                                 Capacity
	}{
		{
			name: "host alone", maxSize: 6, hostSeats: 4, acceptedPeople: 0, driverSeats: 0,
			want: Capacity{
				PeopleCount:  1,
				SeatCapacity: 4,
				SeatsShort:   0,
				SpotsLeft:    3,
				WaitingCount: 0,
			},
		},
		{
			name: "seats bind", maxSize: 12, hostSeats: 4, acceptedPeople: 3, driverSeats: 0, waiting: 0,
			want: Capacity{PeopleCount: 4, SeatCapacity: 4, SeatsShort: 0, SpotsLeft: 0, WaitingCount: 0},
		},
		{
			name: "seats short", maxSize: 12, hostSeats: 2, acceptedPeople: 4, driverSeats: 0, waiting: 0,
			want: Capacity{
				PeopleCount:  5,
				SeatCapacity: 2,
				SeatsShort:   3,
				SpotsLeft:    0,
				WaitingCount: 0,
			},
		},
		{
			name: "driver opens seat", maxSize: 12, hostSeats: 2, acceptedPeople: 3, driverSeats: 4, waiting: 0,
			want: Capacity{
				PeopleCount:  4,
				SeatCapacity: 6,
				SeatsShort:   0,
				SpotsLeft:    2,
				WaitingCount: 0,
			},
		},
		{
			name:    "max_size binds",
			maxSize: 5, hostSeats: 4, acceptedPeople: 3, driverSeats: 4, waiting: 0,
			want: Capacity{
				PeopleCount:  4,
				SeatCapacity: 8,
				SeatsShort:   0,
				SpotsLeft:    1,
				WaitingCount: 0,
			},
		},
		{
			name: "waiting passes through", maxSize: 6, hostSeats: 4, acceptedPeople: 0, driverSeats: 0, waiting: 5,
			want: Capacity{
				PeopleCount:  1,
				SeatCapacity: 4,
				SeatsShort:   0,
				SpotsLeft:    3,
				WaitingCount: 5,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := &Outing{MaxSize: tc.maxSize, HostSeats: tc.hostSeats}
			got := capacityOf(o, tc.acceptedPeople, tc.driverSeats, tc.waiting)
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}

		})
	}
}
