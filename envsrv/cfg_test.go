package envsrv

import (
	"net/http"
	"testing"
)

func TestGojiRoutesBuiltProperly(t *testing.T) {
	routes := []Node{
		Node{Name: "omc"},
		Node{Name: "dst"},
		Node{Name: "gpct"},
		Node{Name: "piaacmc"},
		Node{Name: "env", Parent: "omc"},
		Node{Name: "vacuum", Parent: "env"},
	}

	mux := BuildNetwork()

	req, _ := http.NewRequest("GET", "/omc/env/vacuum")
}
