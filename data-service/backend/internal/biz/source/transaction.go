package source

import "context"

type Store interface {
	List(context.Context, bool) ([]Source, error)
	InTransaction(context.Context, func(Transaction) error) error
}

type Transaction interface {
	Lock(context.Context) error
	List(context.Context) ([]Source, error)
	Insert(context.Context, Source) (Source, error)
	Update(context.Context, Source) (Source, error)
	Delete(context.Context, string) error
}
