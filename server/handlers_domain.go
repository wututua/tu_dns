package server

import (
	"strconv"

	"tudns/dns"
	"tudns/middleware"
	"tudns/record"
	"tudns/response"

	"github.com/gin-gonic/gin"
)

func (a *App) handlePublicDomains(c *gin.Context) {
	items, err := a.domain.ListPublic()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleDNSProviders(c *gin.Context) {
	response.OK(c, dns.List())
}

func (a *App) handleCreateBundle(c *gin.Context) {
	var req record.BundleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	uid := middleware.CurrentUserID(c)
	res, err := a.record.CreateBundle(uid, req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	a.admin.WriteLog(uid, 0, "create_bundle", "subdomain", strconv.FormatUint(uint64(res.Subdomain.ID), 10), c.ClientIP(), res.Subdomain.FullDomain)
	response.OK(c, res)
}

func (a *App) handleMySubdomains(c *gin.Context) {
	items, err := a.record.ListSubdomains(middleware.CurrentUserID(c))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleDeleteSubdomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	uid := middleware.CurrentUserID(c)
	role, _ := c.Get(middleware.CtxRole)
	if err := a.record.DeleteSubdomain(uid, uint(id), role == "admin"); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) handleMyRecords(c *gin.Context) {
	items, err := a.record.ListByUser(middleware.CurrentUserID(c))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleAddRecord(c *gin.Context) {
	var req record.AddRecordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	uid := middleware.CurrentUserID(c)
	role, _ := c.Get(middleware.CtxRole)
	rec, charged, err := a.record.AddRecord(uid, req, role == "admin")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"record": rec, "charged": charged})
}

func (a *App) handleUpdateRecord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req record.UpdateRecordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	uid := middleware.CurrentUserID(c)
	role, _ := c.Get(middleware.CtxRole)
	rec, err := a.record.UpdateRecord(uid, uint(id), req, role == "admin")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, rec)
}

func (a *App) handleDeleteRecord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	uid := middleware.CurrentUserID(c)
	role, _ := c.Get(middleware.CtxRole)
	if err := a.record.DeleteRecord(uid, uint(id), role == "admin"); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}
