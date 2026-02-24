package web

import (
	"net/http"
	"sort"
	"strconv"
	"text/template"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

type ZoneDetails struct {
	ZoneName  string
	RoomCount int
}

func roomsIndex(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(configs.GetFilePathsConfig().AdminHtml.String()+"/_header.html", configs.GetFilePathsConfig().AdminHtml.String()+"/rooms/index.html", configs.GetFilePathsConfig().AdminHtml.String()+"/_footer.html")
	if err != nil {
		mudlog.Error("HTML Template", "error", err)
	}

	qsp := r.URL.Query()

	filterType := qsp.Get(`filter-type`)

	type shortRoomInfo struct {
		RoomId          int
		RoomZone        string
		ZoneRoot        bool
		RoomTitle       string
		IsBank          bool
		IsStorage       bool
		IsCharacterRoom bool
		IsSkillTraining bool
		HasContainer    bool
		IsPvp           bool
	}

	allZones := []ZoneDetails{}
	allRooms := []shortRoomInfo{}
	zoneCounter := map[string]int{}

	for _, rId := range rooms.GetAllRoomIds() {
		if room := rooms.LoadRoom(rId); room != nil {

			if _, ok := zoneCounter[room.Zone]; !ok {

				zoneCounter[room.Zone] = 0
				allZones = append(allZones, ZoneDetails{
					ZoneName:  room.Zone,
					RoomCount: 0,
				})
			}
			zoneCounter[room.Zone] = zoneCounter[room.Zone] + 1

			if filterType != `*` && filterType != room.Zone {
				continue
			}

			hasContainer := false

			for _, cInfo := range room.Containers {
				if cInfo.DespawnRound == 0 {
					hasContainer = true
					break
				}
			}

			rootRoomId := 0
			if zCfg := rooms.GetZoneConfig(room.Zone); zCfg != nil {
				rootRoomId = zCfg.RoomId
			}

			allRooms = append(allRooms, shortRoomInfo{
				RoomId:          room.RoomId,
				RoomZone:        room.Zone,
				ZoneRoot:        rootRoomId == room.RoomId,
				RoomTitle:       room.Title,
				IsBank:          room.IsBank,
				IsStorage:       room.IsStorage,
				IsCharacterRoom: room.IsCharacterRoom,
				IsSkillTraining: len(room.SkillTraining) > 0,
				HasContainer:    hasContainer,
				IsPvp:           room.IsPvp(),
			})
		}
	}

	for i, zInfo := range allZones {
		zInfo.RoomCount = zoneCounter[zInfo.ZoneName]
		allZones[i] = zInfo
	}

	sort.SliceStable(allRooms, func(i, j int) bool {

		if allRooms[i].RoomZone != allRooms[j].RoomZone {
			return allRooms[i].RoomZone < allRooms[j].RoomZone
		}

		if allRooms[i].ZoneRoot {
			return true
		} else if allRooms[j].ZoneRoot {
			return false
		}

		return allRooms[i].RoomId < allRooms[j].RoomId
	})

	sort.SliceStable(allZones, func(i, j int) bool {
		return allZones[i].ZoneName < allZones[j].ZoneName
	})

	tplData := map[string]any{
		`Zones`:      allZones,
		`Rooms`:      allRooms,
		`FilterType`: filterType,
	}

	if err := tmpl.Execute(w, tplData); err != nil {
		mudlog.Error("HTML Execute", "error", err)
	}

}

func roomData(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.New("room.data.html").Funcs(funcMap).ParseFiles(configs.GetFilePathsConfig().AdminHtml.String() + "/rooms/room.data.html")
	if err != nil {
		mudlog.Error("HTML Template", "error", err)
	}

	urlVals := r.URL.Query()

	roomIdInt, _ := strconv.Atoi(urlVals.Get(`roomid`))

	roomInfo := rooms.LoadRoom(roomIdInt)

	tplData := map[string]any{}
	tplData[`roomInfo`] = roomInfo

	buffSpecs := []buffs.BuffSpec{}
	for _, buffId := range buffs.GetAllBuffIds() {
		if b := buffs.GetBuffSpec(buffId); b != nil {
			if b.Name == `empty` {
				continue
			}
			buffSpecs = append(buffSpecs, *b)
		}
	}
	sort.SliceStable(buffSpecs, func(i, j int) bool {
		return buffSpecs[i].BuffId < buffSpecs[j].BuffId
	})
	tplData[`buffSpecs`] = buffSpecs

	allBiomes := rooms.GetAllBiomes()
	sort.SliceStable(allBiomes, func(i, j int) bool {
		return allBiomes[i].Name < allBiomes[j].Name
	})
	tplData[`biomes`] = allBiomes

	allSkillNames := []string{}
	for _, name := range skills.GetAllSkillNames() {
		allSkillNames = append(allSkillNames, string(name))
	}
	sort.SliceStable(allSkillNames, func(i, j int) bool {
		return allSkillNames[i] < allSkillNames[j]
	})
	tplData[`allSkillNames`] = allSkillNames

	tplData[`allSlotTypes`] = characters.GetAllSlotTypes()

	mapDirections := []string{}

	for _, name := range mapper.GetDirectionDeltaNames() {
		mapDirections = append(mapDirections, name)
	}
	sort.SliceStable(mapDirections, func(i, j int) bool {
		return mapDirections[i] < mapDirections[j]
	})
	tplData[`mapDirections`] = mapDirections

	mutSpecs := mutators.GetAllMutatorSpecs()
	sort.SliceStable(mutSpecs, func(i, j int) bool {
		return mutSpecs[i].MutatorId < mutSpecs[j].MutatorId
	})
	tplData[`mutSpecs`] = mutSpecs

	if err := tmpl.Execute(w, tplData); err != nil {
		mudlog.Error("HTML Execute", "error", err)
	}

}
