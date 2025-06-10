package ast

// Visitor processes AST nodes and can transform them. Nodes are visited in
// post-order (children first).
type Visitor interface {
	Visit(Expr) (Expr, error)
}

func walkChild(child *Expr, v Visitor) error {
	newChild, err := (*child).Walk(v)
	if err != nil {
		return err
	}
	*child = newChild

	return nil
}

func walkChildren(children []Expr, v Visitor) error {
	for i := range children {
		if err := walkChild(&children[i], v); err != nil {
			return err
		}
	}

	return nil
}
