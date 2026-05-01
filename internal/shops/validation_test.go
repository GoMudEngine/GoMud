package shops

import (
	"strings"
	"testing"
)

// fakeMob satisfies the minimal interface ValidateShopMobTags needs.
type fakeMob struct {
	mobId            int
	name             string
	zone             string
	hasShop          bool
	isCrafter        bool
	shopCraftSupport string
}

func (f fakeMob) GetMobId() int            { return f.mobId }
func (f fakeMob) GetName() string          { return f.name }
func (f fakeMob) GetZone() string          { return f.zone }
func (f fakeMob) HasShop() bool            { return f.hasShop }
func (f fakeMob) IsCrafter() bool          { return f.isCrafter }
func (f fakeMob) GetShopCraftSupport() string { return f.shopCraftSupport }

func TestValidateShopMobTags_AllValid(t *testing.T) {
	mobs := []ShopBearingMob{
		fakeMob{mobId: 1, hasShop: true, shopCraftSupport: CraftSupportBlacksmithing},
		fakeMob{mobId: 2, isCrafter: true, shopCraftSupport: CraftSupportGeneral},
	}
	if err := ValidateShopMobTags(mobs); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateShopMobTags_MissingTag(t *testing.T) {
	mobs := []ShopBearingMob{
		fakeMob{mobId: 99, name: "broken", zone: "z", hasShop: true, shopCraftSupport: ""},
	}
	err := ValidateShopMobTags(mobs)
	if err == nil {
		t.Fatal("expected error for missing tag, got nil")
	}
	if !strings.Contains(err.Error(), "99") || !strings.Contains(err.Error(), "broken") {
		t.Errorf("error should reference the offending mob id and name; got: %v", err)
	}
}

func TestValidateShopMobTags_InvalidTag(t *testing.T) {
	mobs := []ShopBearingMob{
		fakeMob{mobId: 100, hasShop: true, shopCraftSupport: "knitting"},
	}
	err := ValidateShopMobTags(mobs)
	if err == nil || !strings.Contains(err.Error(), "knitting") {
		t.Errorf("expected error mentioning invalid tag; got: %v", err)
	}
}

func TestValidateShopMobTags_NonShopMobsIgnored(t *testing.T) {
	mobs := []ShopBearingMob{
		fakeMob{mobId: 5, hasShop: false, isCrafter: false, shopCraftSupport: ""},
	}
	if err := ValidateShopMobTags(mobs); err != nil {
		t.Fatalf("non-shop mobs should not be validated; got: %v", err)
	}
}
