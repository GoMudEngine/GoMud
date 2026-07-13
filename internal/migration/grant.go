package migration

// grant.go — the seed a classified player receives on migration (spec §5). Even
// the heaviest veterans arrive NEAR their ceiling, not at an apex; progression is
// use-based so a modest seed is a starting point they regrow. Ranks are
// PROVISIONAL, retuned with the NPC kits in the 6e balance pass.

// SeedForCluster returns the mutation seed granted for a classification. Every id
// is a live post-nuke mutation. Empty map for an unknown classification.
func SeedForCluster(cluster string) map[string]int {
	switch cluster {
	case "manifester":
		// Broodmaster keystones, just short of the brood-mother apex (meirok).
		return map[string]int{"symbiotic-bond": 1, "hive-mind": 1}
	case "ethereal":
		return map[string]int{"ether-gland": 1, "second-sight": 1}
	case "ravener":
		return map[string]int{"rending-claws": 1, "raptor-legs": 1}
	case "colossus":
		return map[string]int{"dense-muscles": 1, "titan-growth": 1}
	case "stalker":
		return map[string]int{"padded-soles": 1, "compound-eyes": 1}
	case "zealot":
		return map[string]int{"commanding-presence": 1, "zealous-conviction": 1}
	case "chrysifier":
		// Crafter keystones; the apex is still earned through play (Duard-class).
		return map[string]int{"provident-hands": 1, "walking-chrysalis": 1}
	case "generalist":
		// One Center entry keystone at rank 1; grow through play.
		return map[string]int{"keen-senses": 1}
	case "admin":
		// All-access: one entry keystone per cluster so the admin can test the
		// whole system from a single character (spec §6).
		return map[string]int{
			"dense-muscles": 1, "thick-hide": 1, "rending-claws": 1, "padded-soles": 1,
			"ether-gland": 1, "symbiotic-bond": 1, "commanding-presence": 1,
			"sticky-secretion": 1, "quicksilver-nerves": 1, "provident-hands": 1, "keen-senses": 1,
		}
	}
	return nil
}
