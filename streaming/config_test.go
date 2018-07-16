/* Copyright (c) 2018 Gregor Riepl
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package streaming

import (
	// 	"encoding/json"
	"reflect"
	"testing"
)

func TestArrayRange(t *testing.T) {
	vals := []struct{ X int }{
		{0},
	}
	for i := range vals {
		val := &vals[i]
		val.X = 1
	}
	if vals[0].X == 0 {
		t.Errorf("Invalid value")
	}
}

func TestConfig01(t *testing.T) {
	t01 := &Configuration{
		Listen:       "localhost:http",
		Timeout:      0,
		Reconnect:    10,
		InputBuffer:  1000,
		OutputBuffer: 400,
		NoStats:      false,
	}
	r01 := DefaultConfiguration()
	if !reflect.DeepEqual(t01, r01) {
		t.Errorf("Default configuration does not match test case")
	}
}

func TestConfig02(t *testing.T) {
	t02 := &Configuration{
		Listen: "testhost:9999",
	}
	c02 := `{
		"listen": "testhost:9999"
	}`
	r02, e02 := LoadConfigurationBytes([]byte(c02))
	if e02 != nil || t02.Listen != r02.Listen {
		t.Errorf("Variable loaded from JSON does not match expected result")
	}
}

func TestConfig03(t *testing.T) {
	t03 := DefaultConfiguration()
	t03.Listen = "testhost:9999"
	c03 := `{
		"listen": "testhost:9999"
	}`
	r03, e03 := LoadConfigurationBytes([]byte(c03))
	if e03 != nil || !reflect.DeepEqual(t03, r03) {
		t.Logf("t03: %v", t03)
		t.Logf("r03: %v", r03)
		t.Errorf("Loaded JSON configuration does not match default configuration plus variable")
	}
}

func TestConfig04(t *testing.T) {
	t04 := DefaultConfiguration()
	t04.Resources = []Resource{
		Resource{
			Type: "stream",
			Remotes: []string{
				"t04",
			},
		},
	}
	c04 := `{
		"resources": [
			{
				"type": "stream",
				"remote": "t04"
			}
		]
	}`
	r04, e04 := LoadConfigurationBytes([]byte(c04))
	if e04 != nil || !reflect.DeepEqual(t04, r04) {
		t.Logf("t04: %v", t04)
		t.Logf("r04: %v", r04)
		t.Logf("e04: %v", e04)
		t.Errorf("Single remote for stream not parsed correctly")
	}

	t04b := DefaultConfiguration()
	t04b.Resources = []Resource{
		Resource{
			Type:   "api",
			Remote: "t04b",
		},
	}
	c04b := `{
		"resources": [
			{
				"type": "api",
				"remote": "t04b"
			}
		]
	}`
	r04b, e04b := LoadConfigurationBytes([]byte(c04b))
	if e04b != nil || !reflect.DeepEqual(t04b, r04b) {
		t.Logf("t04b: %v", t04b)
		t.Logf("r04b: %v", r04b)
		t.Logf("e04b: %v", e04b)
		t.Errorf("Single remote for API not parsed correctly")
	}
}

func TestConfig05(t *testing.T) {
	t05 := DefaultConfiguration()
	t05.Resources = []Resource{
		Resource{
			Authentication: Authentication{
				Type: "basic",
				Users: []string{
					"t05",
				},
			},
		},
	}
	c05 := `{
		"resources": [
			{
				"authentication": {
					"type": "basic",
					"user": "t05"
				}
			}
		]
	}`
	r05, e05 := LoadConfigurationBytes([]byte(c05))
	if e05 != nil || !reflect.DeepEqual(t05, r05) {
		t.Logf("t05: %v", t05)
		t.Logf("r05: %v", r05)
		t.Logf("e05: %v", e05)
		t.Errorf("User list not parsed correctly")
	}
}

func TestConfig06(t *testing.T) {
	t06 := DefaultConfiguration()
	t06.Notifications = []Notification{
		{
			Authentication: Authentication{
				Type: "basic",
				User: "t06",
			},
		},
	}
	c06 := `{
		"notifications": [
			{
				"authentication": {
					"type": "basic",
					"users": [
						"t06"
					]
				}
			}
		]
	}`
	r06, e06 := LoadConfigurationBytes([]byte(c06))
	if e06 != nil || !reflect.DeepEqual(t06, r06) {
		t.Logf("t06: %v", t06)
		t.Logf("r06: %v", r06)
		t.Logf("e06: %v", e06)
		t.Errorf("Notification user not parsed correctly")
	}
}
