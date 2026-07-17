package server

import (
	"strconv"

	"tudns/dns"

	"tudns/record"

	"github.com/gin-gonic/gin"
)

func (a *App) handlePublicDomains(c *gin.Context) {
	items, err := a.domain.ListPublic()
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}

func (a *App) handleDNSProviders(c *gin.Context) {
	OK(c, dns.List())
}

func (a *App) handleCreateBundle(c *gin.Context) {
	var req record.BundleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	uid := CurrentUserID(c)
	res, err := a.record.CreateBundle(uid, req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	a.admin.WriteLog(uid, 0, "create_bundle", "subdomain", strconv.FormatUint(uint64(res.Subdomain.ID), 10), c.ClientIP(), res.Subdomain.FullDomain)
	OK(c, res)
}

func (a *App) handleMySubdomains(c *gin.Context) {
	items, err := a.record.ListSubdomains(CurrentUserID(c))
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}

func (a *App) handleDeleteSubdomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	uid := CurrentUserID(c)
	role, _ := c.Get(CtxRole)
	if err := a.record.DeleteSubdomain(uid, uint(id), role == "admin"); err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}

func (a *App) handleMyRecords(c *gin.Context) {
	items, err := a.record.ListByUser(CurrentUserID(c))
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}

func (a *App) handleAddRecord(c *gin.Context) {
	var req record.AddRecordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	uid := CurrentUserID(c)
	role, _ := c.Get(CtxRole)
	rec, charged, err := a.record.AddRecord(uid, req, role == "admin")
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, gin.H{"record": rec, "charged": charged})
}

func (a *App) handleUpdateRecord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req record.UpdateRecordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	uid := CurrentUserID(c)
	role, _ := c.Get(CtxRole)
	rec, err := a.record.UpdateRecord(uid, uint(id), req, role == "admin")
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, rec)
}

func (a *App) handleDeleteRecord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	uid := CurrentUserID(c)
	role, _ := c.Get(CtxRole)
	if err := a.record.DeleteRecord(uid, uint(id), role == "admin"); err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}
