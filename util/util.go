package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// MapSize is the size of the subuid mapping
const MapSize = 65536

// PrettyPrintStruct prints a struct as info level by default
// unless a different logging level function is passed in
func PrettyPrintStruct(data interface{}) {
	jsonified, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		err = fmt.Errorf("failed to pretty print json: %v", err)
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("%+v", (string(jsonified)))
}

// UnmarshalFile reads a json file into a struct
func UnmarshalFile(filepath string, dst interface{}) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return errors.WithStack(err)
	}
	err = json.Unmarshal(data, dst)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// GetSubid looks up the subuid and subgid for the given user
func GetSubid(userObj *user.User) (int, int, error) {
	subuidFile, err := os.Open("/etc/subuid")
	if err != nil {
		return -1, -1, err
	}
	defer subuidFile.Close()
	subuid, err := parseSubidFile(subuidFile, userObj.Username, userObj.Uid)
	if err != nil {
		return -1, -1, err
	}

	subgidFile, err := os.Open("/etc/subgid")
	if err != nil {
		return -1, -1, err
	}
	defer subgidFile.Close()
	subgid, err := parseSubidFile(subgidFile, userObj.Username, userObj.Gid)
	if err != nil {
		return -1, -1, err
	}

	return subuid, subgid, nil
}

func parseSubidFile(subidFile *os.File, name string, id string) (int, error) {
	scanner := bufio.NewScanner(subidFile)
	for scanner.Scan() {
		err := scanner.Err()
		if err != nil {
			return -1, err
		}

		parts := strings.Split(scanner.Text(), ":")
		if len(parts) != 3 {
			return -1, fmt.Errorf("invalid /etc/sub[gu]id file")
		}
		if parts[0] == name || parts[0] == id {
			size, err := strconv.Atoi(parts[2])
			if err != nil {
				return -1, err
			}
			if size >= MapSize {
				subid, err := strconv.Atoi(parts[1])
				if err != nil {
					return -1, err
				}
				return subid, nil
			}
		}
	}
	return -1, fmt.Errorf("no matching sub[gu]id found for user %s", name)
}
