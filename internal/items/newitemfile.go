package items

import (
	"errors"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
)

func CreateNewItemFile(newItemInfo ItemSpec) (int, error) {

	newItemInfo.ItemId = getNextItemId(newItemInfo.Type)
	if newItemInfo.ItemId == 0 {
		return 0, errors.New(`Could not find a new item id to assign.`)
	}

	CanonicalizeItemNames(&newItemInfo)

	if err := newItemInfo.Validate(); err != nil {
		return 0, err
	}

	//
	// Before saving, lets zero out the damage info we don't need written.
	// (It can be derrived from the diceroll which is set.)
	//
	newItemInfo.Damage.Attacks = 0
	newItemInfo.Damage.DiceCount = 0
	newItemInfo.Damage.SideCount = 0

	//
	// Save the item
	//
	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}

	if err := fileloader.SaveFlatFile[*ItemSpec](configs.GetFilePathsConfig().DataFiles.String()+`/items`, &newItemInfo, saveModes...); err != nil {
		return 0, err
	}

	// Save to in-memory cache
	items[newItemInfo.Id()] = &newItemInfo

	return newItemInfo.Id(), nil
}

func getNextItemId(t ItemType) int {

	rangeMin := 0
	rangeMax := 0

	tString := string(t)
	for _, iType := range ItemTypes() {

		if iType.Type == tString {
			rangeMin = iType.MinItemId
			rangeMax = iType.MaxItemId
			break
		}

	}

	if rangeMin == 0 && rangeMax == 0 {
		return 0
	}

	lowestFreeId := 1
	for _, iSpec := range items {

		if iSpec.ItemId < rangeMin || iSpec.ItemId > rangeMax {
			continue
		}

		if iSpec.ItemId >= lowestFreeId {
			lowestFreeId = iSpec.ItemId + 1
		}
	}

	if lowestFreeId < rangeMin || lowestFreeId > rangeMax {
		return 0
	}

	return lowestFreeId
}
