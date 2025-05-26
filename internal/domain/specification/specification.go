package specification

import (
	"context"
)

// Specification defines the interface for query specifications
type Specification interface {
	// IsSatisfiedBy checks if the specification is satisfied by the given object
	IsSatisfiedBy(candidate interface{}) bool
	// ToSQL converts the specification to SQL WHERE clause and parameters
	ToSQL() (string, []interface{})
}

// CompositeSpecification provides AND, OR, and NOT operations
type CompositeSpecification interface {
	Specification
	And(other Specification) Specification
	Or(other Specification) Specification
	Not() Specification
}

// BaseSpecification provides common specification functionality
type BaseSpecification struct{}

// And creates an AND specification
func (s BaseSpecification) And(other Specification) Specification {
	return &andSpecification{
		left:  s,
		right: other,
	}
}

// Or creates an OR specification
func (s BaseSpecification) Or(other Specification) Specification {
	return &orSpecification{
		left:  s,
		right: other,
	}
}

// Not creates a NOT specification
func (s BaseSpecification) Not() Specification {
	return &notSpecification{
		spec: s,
	}
}

// andSpecification represents an AND combination of specifications
type andSpecification struct {
	left  Specification
	right Specification
}

func (s *andSpecification) IsSatisfiedBy(candidate interface{}) bool {
	return s.left.IsSatisfiedBy(candidate) && s.right.IsSatisfiedBy(candidate)
}

func (s *andSpecification) ToSQL() (string, []interface{}) {
	leftSQL, leftParams := s.left.ToSQL()
	rightSQL, rightParams := s.right.ToSQL()
	
	sql := "(" + leftSQL + " AND " + rightSQL + ")"
	params := append(leftParams, rightParams...)
	
	return sql, params
}

// orSpecification represents an OR combination of specifications
type orSpecification struct {
	left  Specification
	right Specification
}

func (s *orSpecification) IsSatisfiedBy(candidate interface{}) bool {
	return s.left.IsSatisfiedBy(candidate) || s.right.IsSatisfiedBy(candidate)
}

func (s *orSpecification) ToSQL() (string, []interface{}) {
	leftSQL, leftParams := s.left.ToSQL()
	rightSQL, rightParams := s.right.ToSQL()
	
	sql := "(" + leftSQL + " OR " + rightSQL + ")"
	params := append(leftParams, rightParams...)
	
	return sql, params
}

// notSpecification represents a NOT specification
type notSpecification struct {
	spec Specification
}

func (s *notSpecification) IsSatisfiedBy(candidate interface{}) bool {
	return !s.spec.IsSatisfiedBy(candidate)
}

func (s *notSpecification) ToSQL() (string, []interface{}) {
	sql, params := s.spec.ToSQL()
	return "NOT " + sql, params
}

// Repository interface that supports specifications
type Repository interface {
	// FindBySpecification finds entities matching the specification
	FindBySpecification(ctx context.Context, spec Specification) ([]interface{}, error)
	// CountBySpecification counts entities matching the specification
	CountBySpecification(ctx context.Context, spec Specification) (int64, error)
}