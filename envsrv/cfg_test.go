package envsrv

import "testing"

func TestGojiRoutesBuiltProperly(t *testing.T) {
	routes := []Node{
		Node{Name: "omc"},
		Node{Name: "dst"},
		Node{Name: "gpct"},
		Node{Name: "piaacmc"},
		Node{Name: "env", Parent: "omc"},
		Node{Name: "vacuum", Parent: "env"},
	}
}
