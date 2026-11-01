// src/internal/outing/broadcast_test.go
package outing

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Every roster or request transition, plus Update and Cancel, emits one
// outing.changed carrying the outing id (design E2, E4). Comments don't (E8).
// One service and fake for all rows; each row gets its own outing and counts
// only the events that name it, so rows can't see each other's broadcasts.
func Test_OutingChanged_Table(t *testing.T) {
	svc, f, _, fb := newTestService(t)
	ctx := context.Background()
	host, hiker := uuid.New(), uuid.New()

	cases := []struct {
		name string
		act  func(o *Outing) error
		want int
	}{
		{name: "Update", want: 1, act: func(o *Outing) error {
			_, err := svc.Update(ctx, host, o.ID, UpdateInput{Title: ptr("renamed")})
			return err
		}},
		{name: "Cancel", want: 1, act: func(o *Outing) error {
			return svc.Cancel(ctx, host, o.ID)
		}},
		{name: "RequestJoin", want: 1, act: func(o *Outing) error {
			_, err := svc.RequestJoin(ctx, hiker, o.ID, JoinInput{Role: RoleRider})
			return err
		}},
		{name: "EditJoinRequest", want: 1, act: func(o *Outing) error {
			req := seedJoinRequest(o.ID, hiker, RequestStatusRequested, RoleRider, f, 0)
			_, err := svc.EditJoinRequest(ctx, hiker, req.ID, EditJoinRequestInput{Guests: ptr(1)})
			return err
		}},
		{name: "Withdraw", want: 1, act: func(o *Outing) error {
			seedJoinRequest(o.ID, hiker, RequestStatusRequested, RoleRider, f, 0)
			return svc.Withdraw(ctx, hiker, o.ID)
		}},
		{name: "Accept", want: 1, act: func(o *Outing) error {
			req := seedJoinRequest(o.ID, hiker, RequestStatusRequested, RoleRider, f, 0)
			return svc.Accept(ctx, host, req.ID)
		}},
		{name: "Decline", want: 1, act: func(o *Outing) error {
			req := seedJoinRequest(o.ID, hiker, RequestStatusRequested, RoleRider, f, 0)
			return svc.Decline(ctx, host, req.ID)
		}},
		{name: "RemoveMember", want: 1, act: func(o *Outing) error {
			req := seedJoinRequest(o.ID, hiker, RequestStatusAccepted, RoleRider, f, 0)
			return svc.RemoveMember(ctx, host, req.ID)
		}},
		{name: "RestoreRequest", want: 1, act: func(o *Outing) error {
			req := seedJoinRequest(o.ID, hiker, RequestStatusDeclined, RoleRider, f, 0)
			return svc.RestoreRequest(ctx, host, req.ID)
		}},
		{name: "AddComment", want: 0, act: func(o *Outing) error {
			seedJoinRequest(o.ID, hiker, RequestStatusAccepted, RoleRider, f, 0)
			seedMember(hiker, "mahmut", "experienced", f)
			_, err := svc.AddComment(ctx, hiker, o.ID, "Halo!", nil)
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := seedOuting(8, 4, StatusOpen, host, f)
			if err := tc.act(o); err != nil {
				t.Fatalf("%s failed: %v", tc.name, err)
			}
			got := 0
			for _, e := range fb.broadcasts() {
				if e.Type == "outing.changed" && strings.Contains(e.Data, o.ID.String()) {
					got++
				}
			}
			if got != tc.want {
				t.Errorf("want %d outing.changed for this outing, got %d", tc.want, got)
			}
		})
	}
}
