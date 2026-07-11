# South Road + Amber Valley — Coordinate Map (spatial source of truth)

Every room's exact `(x,y,z)` and exits. Room-authoring tasks MUST place rooms at
these coords with these exits (reciprocity already worked out here). Geometry is
deliberately simple (straight spine + compact grid + short spurs) to stay
cartcheck-clean. The boot-time `ValidateZoneConsistency` (mode=panic) is the hard
verifier.

**Anchor:** Marches Spur Road room **4014** at `(x=-8, y=-13, z=0)`. South =
decreasing y. The southern branch is a fresh region; nothing else is built south
of the crossroads.

**Exit notation:** `N`=north `S`=south `E`=east `W`=west `U`=up `D`=down. A
`*zone:X*` tag marks a cross-zone exit (must be annotated in the YAML). All exits
listed are reciprocal with the named room.

## Zone: South Road (`south_road`, 6040–6054) — straight N–S spine, x=-8

| Room | coord (x,y,z) | exits | role |
|------|---------------|-------|------|
| 6040 | -8,-14,0 | N→4014 *zone:Marches Spur Road*, S→6041 | The Southern Verge — leaving the crossroads, heat rising off the road |
| 6041 | -8,-15,0 | N→6040, S→6042 | The Drovers' Descent — road bends south & down, valley hazed below |
| 6042 | -8,-16,0 | N→6041, S→6043 | The Drying Mile — orchard country thinning to yellow scrub |
| 6043 | -8,-17,0 | N→6042, S→6044 | Orchard's End — last fruit-trees give out; merchants pass northbound |
| 6044 | -8,-18,0 | N→6043, S→6045 | The Lake & Ladle — the waypoint inn (innkeeper anchor; hearth/sign nouns) |
| 6045 | -8,-19,0 | N→6044, S→6046 | The Shepherd's Reach — pasture & vantage (shepherd anchor) |
| 6046 | -8,-20,0 | N→6045, S→6047 | The Long Furlong — open road, sheep-bells on the wind |
| 6047 | -8,-21,0 | N→6046, S→6048 | The Valley Overlook — the whole warm valley opens below (vista noun) |
| 6048 | -8,-22,0 | N→6047, S→6049 | The Warm Descent — air thick with sun-baked earth |
| 6049 | -8,-23,0 | N→6048, S→6050 | The First Orchards — irrigated green returns; smell of ripening fruit |
| 6050 | -8,-24,0 | N→6049, S→6051 | Irrigation Row — channels stitched across farmland (Stage B) |
| 6051 | -8,-25,0 | N→6050, S→6052 | The Vineward Bend — vines on the slopes |
| 6052 | -8,-26,0 | N→6051, S→6053 | The Dryside Track — farmland going dry on the east side |
| 6053 | -8,-27,0 | N→6052, S→6054 | The Dryside Farmstead — **dried channel** noun (Water-Dispute breadcrumb) |
| 6054 | -8,-28,0 | N→6053, S→6055 *zone:Amber Valley* | Valley Gate — South Road's end, Amber Valley's threshold |

## Zone: Amber Valley (`amber_valley`, 6055–6089)

### Town Center (6055–6064)

| Room | coord | exits | role |
|------|-------|-------|------|
| 6055 | -8,-29,0 | N→6054 *zone:South Road*, S→6056 | Valley Road — entering the town |
| 6056 | -8,-30,0 | N→6055, S→6057, E→6058, W→6059 | The Market Square — hub; fruit stalls; farmers grumble here |
| 6057 | -8,-31,0 | N→6056, S→6060 | Pavilion Walk |
| 6058 | -7,-30,0 | W→6056, E→6061, S→6063 | East Market — fruit stalls (forage/flavor) |
| 6059 | -9,-30,0 | E→6056, W→6062 | West Lane — craft lane |
| 6060 | -8,-32,0 | N→6057, S→6065 | The Rite Pavilion — Blooming ceremonies; **records** noun (quest "record" path) |
| 6061 | -6,-30,0 | W→6058 | The Golden Bough Inn — innkeeper (QUEST GIVER) |
| 6062 | -10,-30,0 | E→6059 | The General Store — vendor |
| 6063 | -7,-31,0 | N→6058, S→6064 | Tanner's Row — townsfolk |
| 6064 | -7,-32,0 | N→6063 | The Old Well Yard — townsfolk; the struggling youth (seeded) |

### Residential & Farms (6065–6074)

| Room | coord | exits | role |
|------|-------|-------|------|
| 6065 | -8,-33,0 | N→6060, S→6066 | Orchard Lane |
| 6066 | -8,-34,0 | N→6065, S→6067 | The Woodworker's House — **Davan's father** (anchor) |
| 6067 | -8,-35,0 | N→6066, S→6068 | Vineyard Walk |
| 6068 | -8,-36,0 | N→6067, S→6069, E→6072, W→6073 | The Irrigation Head-Gate — **head-gate** noun; the disputed water |
| 6069 | -8,-37,0 | N→6068, S→6070, E→6085 | Lower Orchard Lane |
| 6070 | -8,-38,0 | N→6069, S→6071 | The Dryside Track |
| 6071 | -8,-39,0 | N→6070, W→6075 | Valley's South Lane — **south frontier stub** (signed dead-end toward River Road, NO exit) |
| 6072 | -7,-36,0 | W→6068, S→6074 | Farmer A's Holding — **Farmer A** (dispute party) |
| 6073 | -9,-36,0 | E→6068 | Farmer B's Holding — **Farmer B** (dispute party) |
| 6074 | -7,-37,0 | N→6072 | The Vineyard — vines (forage) |

### Valley Edges + Cave (6075–6084)

| Room | coord | exits | role |
|------|-------|-------|------|
| 6075 | -9,-39,0 | E→6071, W→6076, S→6084 | Foothill Path |
| 6076 | -10,-39,0 | E→6075, W→6077, N→6078 | Dry Scrub Slope |
| 6077 | -11,-39,0 | E→6076, S→6079 | The Ridge Path |
| 6078 | -10,-38,0 | S→6076 | The Collapsed Channel — **collapsed channel** noun (quest "restore" path) |
| 6079 | -11,-40,0 | N→6077, D→6080 | Cave Mouth |
| 6080 | -11,-40,-1 | U→6079, D→6081 | The Cave — Upper (fauna) |
| 6081 | -11,-40,-2 | U→6080, W→6082 | The Cave — Gallery (fauna) |
| 6082 | -12,-40,-2 | E→6081, D→6083 | The Cave — Pool (fauna) |
| 6083 | -12,-40,-3 | U→6082 | The Cave — Deep (valley-predator depth threat + loot) |
| 6084 | -9,-40,0 | N→6075 | The Old Aqueduct Stub — dry scrub (forage; flavor) |

### The Chrysalis Grove (6085–6089) — SE spur from 6069

| Room | coord | exits | role |
|------|-------|-------|------|
| 6085 | -7,-37,0 | W→6069, E→6086 | Grove Path |
| 6086 | -6,-37,0 | W→6085, E→6087 | The Grove Gate |
| 6087 | -5,-37,0 | W→6086, E→6088, S→6089 | The Chrysalis Grove — reverential heart |
| 6088 | -4,-37,0 | W→6087 | The Markers' Walk — mutation-memorial **markers** |
| 6089 | -5,-38,0 | N→6087 | The Inner Grove — **weathered marker** (SEEDED near-flat orbital symbol) |

## Collision check (done)
All coords are unique. The cave descends via z (-1/-2/-3) so its (x,y) may repeat
the mouth's. Town x∈[-10,-6] y∈[-29,-32]; residential x∈[-9,-7] y∈[-33,-39];
edges/cave x∈[-12,-9] y∈[-38,-40]; grove x∈[-7,-4] y∈[-37,-38]; South Road x=-8
y∈[-14,-28]. No two rooms share an (x,y,z). The boot ValidateZoneConsistency
(mode=panic) is the final gate after each room task.
