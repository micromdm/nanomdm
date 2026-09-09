package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanomdm/mdm"
)

// RetrievePushInfo selects push info for identifiers ids.
//
// Note that we may return fewer results than input. The user of this
// method needs to reconcile that with their requested ids.
func (s *MySQLStorage) RetrievePushInfo(ctx context.Context, ids []string) (map[string]*mdm.Push, error) {
	if len(ids) < 1 {
		return nil, errors.New("no ids provided")
	}

	rows, err := s.q.SelectPushInfo(ctx, ids)
	if err != nil {
		return nil, err
	}

	pushInfos := make(map[string]*mdm.Push, len(rows))
	for _, row := range rows {
		push := &mdm.Push{
			PushMagic: row.PushMagic,
			Topic:     row.Topic,
		}
		if err := push.SetTokenString(row.TokenHex); err != nil {
			return nil, fmt.Errorf("setting push token string for id: %s: %w", row.ID, err)
		}
		pushInfos[row.ID] = push
	}

	return pushInfos, nil
}
