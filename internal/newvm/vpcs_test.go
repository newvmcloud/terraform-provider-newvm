package newvm
package newvm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteVpcWaitsForMembersDeletionAndRetriesBackendError(t *testing.T) {
	memberListCalls := 0
	memberDeleteCalls := 0
	vpcDeleteCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/backend/com.newvm.network/v1/vxlan/vpc-1":
			_, _ = w.Write([]byte(`{"vxlan":{"id":"vpc-1","vxlan":100,"label":"test"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/backend/com.newvm.network/v1/vxlan/member":
			memberListCalls++
			if memberListCalls <= 2 {
				_, _ = w.Write([]byte(`{"members":[{"id":"member-1","orderid":42,"macaddress":"aa:bb:cc:dd:ee:ff","vxlan":100}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"members":[]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/backend/com.newvm.network/v1/vxlan/vpc-1/members":
			memberDeleteCalls++
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read member delete body: %v", err)
			}
			if string(body) != `{"orderid":42}` {
				t.Fatalf("unexpected member delete body: %s", body)
			}
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/backend/com.newvm.network/v1/vxlan/vpc-1":
			vpcDeleteCalls++
			if memberDeleteCalls == 0 {
				t.Fatalf("VPC delete called before member detach")
			}
			if vpcDeleteCalls == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"Failed to trigger VXLAN change"}`))
				return
			}
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := &Client{
		HostURL:    server.URL,
		HTTPClient: server.Client(),
	}

	if err := client.DeleteVpc(context.Background(), "vpc-1"); err != nil {
		t.Fatalf("DeleteVpc returned error: %v", err)
	}

	if memberDeleteCalls != 1 {
		t.Fatalf("expected 1 member delete call, got %d", memberDeleteCalls)
	}

	if memberListCalls < 3 {
		t.Fatalf("expected member list polling before VPC delete, got %d calls", memberListCalls)
	}

	if vpcDeleteCalls != 2 {
		t.Fatalf("expected 2 VPC delete calls, got %d", vpcDeleteCalls)
	}
}