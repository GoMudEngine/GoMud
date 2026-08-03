package configs

import "testing"

func TestCompanionReserveDefaults(t *testing.T) {
	b := GetBalanceConfig()
	if float64(b.CompanionReserveSkillPct) != 0.01 {
		t.Fatalf("CompanionReserveSkillPct = %v, want 0.01", float64(b.CompanionReserveSkillPct))
	}
	if float64(b.CompanionReserveSkillCap) != 0.55 {
		t.Fatalf("CompanionReserveSkillCap = %v, want 0.55", float64(b.CompanionReserveSkillCap))
	}
	if float64(b.CompanionReserveMutPctPerRank) != 0.06 {
		t.Fatalf("CompanionReserveMutPctPerRank = %v, want 0.06", float64(b.CompanionReserveMutPctPerRank))
	}
	if float64(b.CompanionReserveMutCap) != 0.24 {
		t.Fatalf("CompanionReserveMutCap = %v, want 0.24", float64(b.CompanionReserveMutCap))
	}
	if float64(b.CompanionReserveTotalCap) != 0.79 {
		t.Fatalf("CompanionReserveTotalCap = %v, want 0.79", float64(b.CompanionReserveTotalCap))
	}
	if int(b.CompanionSoftCap) != 5 {
		t.Fatalf("CompanionSoftCap = %v, want 5", int(b.CompanionSoftCap))
	}
	if int(b.CompanionSoftCapApex) != 7 {
		t.Fatalf("CompanionSoftCapApex = %v, want 7", int(b.CompanionSoftCapApex))
	}
	if int(b.CompanionReserveDefault) != 280 {
		t.Fatalf("CompanionReserveDefault = %v, want 280", int(b.CompanionReserveDefault))
	}
	if float64(b.CompanionCastingFloorPct) != 0.0 {
		t.Fatalf("CompanionCastingFloorPct = %v, want 0.0", float64(b.CompanionCastingFloorPct))
	}
}
