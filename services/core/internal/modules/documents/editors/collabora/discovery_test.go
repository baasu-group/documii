package collabora

import (
	"strings"
	"testing"
)

func TestParseDiscoveryResolvesAction(t *testing.T) {
	actions, err := parseDiscovery(strings.NewReader(`<wopi-discovery><net-zone><app name="writer"><action name="edit" ext="docx" urlsrc="http://collabora/editor?WOPISrc={WOPISrc}"/></app></net-zone></wopi-discovery>`))
	if err != nil {
		t.Fatal(err)
	}
	if actions[actionKey{"docx", "edit"}] == "" {
		t.Fatal("edit action missing")
	}
}

func TestParseDiscoveryRejectsUnsafeActionURL(t *testing.T) {
	_, err := parseDiscovery(strings.NewReader(`<wopi-discovery><net-zone><app><action name="edit" ext="docx" urlsrc="file:///secret"/></app></net-zone></wopi-discovery>`))
	if err == nil {
		t.Fatal("unsafe discovery action accepted")
	}
}
