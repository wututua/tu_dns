package points

import (
	"testing"

	"tudns/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(&models.User{}, &models.PointsLedger{}); err != nil {
		t.Fatal(err)
	}
	return gdb
}

func TestChargeAndAdjust(t *testing.T) {
	gdb := setupTestDB(t)
	u := models.User{Username: "u1", PasswordHash: "x", Role: models.RoleUser, Status: 1, Points: 100}
	if err := gdb.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(gdb)
	if _, err := svc.Charge(nil, u.ID, 30, "test"); err != nil {
		t.Fatal(err)
	}
	var after models.User
	if err := gdb.First(&after, u.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.Points != 70 {
		t.Fatalf("points=%d", after.Points)
	}
	if _, err := svc.Charge(nil, u.ID, 100, "over"); err == nil {
		t.Fatal("expected insufficient")
	}
}
