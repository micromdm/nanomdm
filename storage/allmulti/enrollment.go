package allmulti

import (
	"context"

	"github.com/micromdm/nanomdm/storage"
)

// QueryEnrollments queries enrollments from the first storage backend.
func (ms *MultiAllStorage) QueryEnrollments(ctx context.Context, req *storage.EnrollmentsQuery) (*storage.EnrollmentsQueryResult, error) {
	val, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return s.QueryEnrollments(ctx, req)
	})
	if val == nil {
		return nil, err
	}
	return val.(*storage.EnrollmentsQueryResult), err
}
