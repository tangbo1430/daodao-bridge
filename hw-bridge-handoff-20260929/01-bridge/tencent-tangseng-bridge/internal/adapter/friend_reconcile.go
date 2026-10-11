package adapter

import (
	"context"
	"fmt"
)

func (c TencentClient) FriendIDs(ctx context.Context, owner string) (map[string]bool, error) {
	all := map[string]bool{}
	start := 0
	for page := 0; page < 100; page++ {
		var r struct {
			Items []struct {
				ID string `json:"To_Account"`
			} `json:"UserDataItem"`
			Complete *int `json:"CompleteFlag"`
			Next     int  `json:"NextStartIndex"`
		}
		if err := c.BusinessRequest(ctx, "sns", "friend_get", map[string]any{"From_Account": owner, "StartIndex": start, "StandardSequence": 0}, &r); err != nil {
			return nil, err
		}
		if r.Complete == nil {
			return nil, fmt.Errorf("friend_get missing completion flag")
		}
		for _, v := range r.Items {
			if v.ID == "" {
				return nil, fmt.Errorf("friend_get missing account")
			}
			all[v.ID] = true
		}
		if *r.Complete != 0 {
			return all, nil
		}
		if r.Next <= start {
			return nil, fmt.Errorf("friend_get cursor stalled")
		}
		start = r.Next
	}
	return nil, fmt.Errorf("friend_get page limit exceeded")
}
