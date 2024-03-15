package app

import (
	"github.com/gin-gonic/gin"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/crux/db"
	"github.com/remiges-tech/crux/db/sqlc-gen"
	"github.com/remiges-tech/crux/server"
	"github.com/remiges-tech/crux/types"
)

// AppList will be responsible for processing the /applist request that comes through as a GET
// The call will return a list of apps in the realm.
func AppList(c *gin.Context, s *service.Service) {
	lh := s.LogHarbour.WithClass("app")
	lh.Log("AppList request received")
	userID, err := server.ExtractUserNameFromJwt(c)
	if err != nil {
		lh.Info().Log("unable to extract userID from token")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(server.MsgId_Missing, server.ERRCode_Token_Data_Missing))
		return
	}

	realmName, err := server.ExtractRealmFromJwt(c)
	if err != nil {
		lh.Info().Log("unable to extract realm from token")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(server.MsgId_Missing, server.ERRCode_Token_Data_Missing))
		return
	}

	var (
		reportCap = []string{"report"}
	)

	isCapable, _ := server.Authz_check(types.OpReq{
		User: userID,
		// The calling user must have `report` capability.
		CapNeeded: reportCap,
	}, false)

	if !isCapable {
		lh.Info().LogActivity("unauthorized user:", userID)
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(server.MsgId_Unauthorized, server.ErrCode_Unauthorized))
		return
	}

	query, ok := s.Dependencies["queries"].(*sqlc.Queries)
	if !ok {
		lh.Debug0().Log("Error while getting query instance from service Dependencies")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(server.MsgId_InternalErr, server.ErrCode_Internal))
		return
	}

	dbResponse, err := query.GetAppList(c, realmName)
	if err != nil {
		lh.Info().Error(err).Log("error while getting app list from db")
		errmsg := db.HandleDatabaseError(err)
		wscutils.SendErrorResponse(c, wscutils.NewResponse(wscutils.ErrorStatus, nil, []wscutils.ErrorMessage{errmsg}))
		return
	}

	lh.Debug0().Log("finished execution of AppList()")
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(map[string]interface{}{"apps": dbResponse}))
}
