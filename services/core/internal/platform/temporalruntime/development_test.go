package temporalruntime

import (
	"context"
	"testing"
)

func TestDevelopmentActivity(t *testing.T) {
	got, err := DevelopmentActivity(context.Background(), DevelopmentInput{Message: "connected"})
	if err != nil || got.Echoed != "connected" {
		t.Fatalf("DevelopmentActivity() = %#v, %v", got, err)
	}
}
