package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"strings"
)

// Android uses the PHP facade. These callbacks also handle older clients; REST
// writes by the configured Tencent administrator are mirror echoes.
func (s *Service) tencentGroupEvent(ctx context.Context, e domain.BusinessEvent) error {
	var echo bool
	var err error
	e, echo, err = s.filterProfileEcho(ctx, e)
	if err != nil || echo {
		return err
	}
	var v struct {
		ID         string  `json:"GroupId"`
		Operator   string  `json:"Operator_Account"`
		ExitType   string  `json:"ExitType"`
		NewOwner   string  `json:"NewOwner_Account"`
		Name       *string `json:"Name"`
		Notice     *string `json:"Notification"`
		FaceURL    *string `json:"FaceUrl"`
		Modified   string  `json:"Modified_Account"`
		Role       *string `json:"Role"`
		NewMembers []struct {
			UID string `json:"Member_Account"`
		} `json:"NewMemberList"`
		ExitMembers []struct {
			UID string `json:"Member_Account"`
		} `json:"ExitMemberList"`
	}
	if err := json.Unmarshal(e.Payload, &v); err != nil {
		return err
	}
	if !strings.HasPrefix(v.ID, "dd_") || v.Operator == s.Tencent.AdminAccount {
		return nil
	}
	if v.Operator == "" {
		return fmt.Errorf("group callback missing operator")
	}
	if _, err := s.owner(ctx, v.Operator); err != nil {
		return err
	}
	req := GroupRequest{Owner: v.Operator, GroupID: v.ID}
	action := "sync"
	switch e.Command {
	case "Group.CallbackAfterNewMemberJoin":
		action = "invite"
		for _, m := range v.NewMembers {
			req.Members = append(req.Members, m.UID)
		}
	case "Group.CallbackAfterMemberExit":
		action = "remove"
		for _, m := range v.ExitMembers {
			req.Members = append(req.Members, m.UID)
		}
		if len(req.Members) == 1 && req.Members[0] == v.Operator {
			action = "exit"
			req.Members = nil
		}
	case "Group.CallbackAfterGroupDestroyed":
		action = "disband"
	case "Group.CallbackAfterChangeGroupOwner":
		action = "update"
		req.Field = "owner"
		req.Value = v.NewOwner
	case "Group.CallbackAfterGroupInfoChanged":
		for _, field := range []struct {
			Key   string
			Value *string
		}{{"name", v.Name}, {"notice", v.Notice}, {"avatar", v.FaceURL}} {
			if field.Value != nil {
				req.Field = field.Key
				req.Value = *field.Value
				if _, err := s.Group(ctx, "update", req); err != nil {
					return s.repairGroupAfterError(ctx, req, err)
				}
			}
		}
		return nil
	case "Group.CallbackAfterMemberFieldChanged":
		if v.Role == nil {
			return nil
		}
		if v.Modified == "" {
			return fmt.Errorf("member callback missing modified account")
		}
		req.Members = []string{v.Modified}
		req.Field = "member"
		if *v.Role == "Admin" {
			req.Field = "manager"
		} else if *v.Role != "Member" {
			return s.repairGroupAfterError(ctx, req, fail(400, "不支持的群成员角色"))
		}
		action = "update"
	}
	_, err = s.Group(ctx, action, req)
	if err != nil {
		return s.repairGroupAfterError(ctx, req, err)
	}
	return nil
}

func (s *Service) repairGroupAfterError(ctx context.Context, req GroupRequest, cause error) error {
	// Restore the Tencent projection from the authoritative Tang state after a
	// permission rejection. Never apply a rejected operation under another user.
	_, repairErr := s.Group(ctx, "sync", GroupRequest{Owner: req.Owner, GroupID: req.GroupID})
	if repairErr != nil {
		return errors.Join(cause, fmt.Errorf("group mirror repair: %w", repairErr))
	}
	return cause
}
