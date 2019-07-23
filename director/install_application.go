package director

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/grahamgilbert/mdmdirector/db"
	"github.com/grahamgilbert/mdmdirector/types"
)

func PostInstallApplicationHandler(w http.ResponseWriter, r *http.Request) {
	// var deviceApplications []types.DeviceInstallApplication
	// var sharedApplications []types.SharedInstallApplication
	var devices []types.Device
	var out types.InstallApplicationPayload

	err := json.NewDecoder(r.Body).Decode(&out)
	if err != nil {
		log.Print(err)
	}

	if out.DeviceUDIDs != nil {
		// Not empty list
		if len(out.DeviceUDIDs) > 0 {
			// Targeting all devices
			if out.DeviceUDIDs[0] == "*" {
				devices = GetAllDevices()
				SaveSharedInstallApplications(out)
				for _, ManifestURL := range out.ManifestURLs {
					// Push these out to existing devices right now now now
					var sharedInstallApplication types.SharedInstallApplication
					sharedInstallApplication.ManifestURL = ManifestURL.URL
					if ManifestURL.BootstrapOnly == false {
						PushSharedInstallApplication(devices, sharedInstallApplication)
					}
				}
			} else {
				for _, item := range out.DeviceUDIDs {
					device := GetDevice(item)
					devices = append(devices, device)
					SaveInstallApplications(devices, out)
				}
				SaveInstallApplications(devices, out)
				for _, ManifestURL := range out.ManifestURLs {
					var installApplication types.DeviceInstallApplication
					installApplication.ManifestURL = ManifestURL.URL
					if ManifestURL.BootstrapOnly == false {
						PushInstallApplication(devices, installApplication)
					}
				}
			}
		}

	} else if out.SerialNumbers != nil {
		if len(out.SerialNumbers) > 0 {
			// Targeting all devices
			if out.SerialNumbers[0] == "*" {
				devices = GetAllDevices()
				SaveSharedInstallApplications(out)
				for _, ManifestURL := range out.ManifestURLs {
					// Push these out to existing devices right now now now
					var sharedInstallApplication types.SharedInstallApplication
					sharedInstallApplication.ManifestURL = ManifestURL.URL
					if ManifestURL.BootstrapOnly == false {
						PushSharedInstallApplication(devices, sharedInstallApplication)
					}
				}
			} else {
				for _, item := range out.SerialNumbers {
					device := GetDeviceSerial(item)
					devices = append(devices, device)
				}
				for _, ManifestURL := range out.ManifestURLs {
					var installApplication types.DeviceInstallApplication
					installApplication.ManifestURL = ManifestURL.URL
					if ManifestURL.BootstrapOnly == false {
						PushInstallApplication(devices, installApplication)
					}
				}
			}
		}

	}
}

// func DeleteProfileHandler(w http.ResponseWriter, r *http.Request) {
// 	var profiles []types.DeviceProfile
// 	var profilesModel types.DeviceProfile
// 	var sharedProfiles []types.SharedProfile
// 	var sharedProfileModel types.SharedProfile
// 	var devices []types.Device
// 	var out types.DeleteProfilePayload

// 	err := json.NewDecoder(r.Body).Decode(&out)
// 	if err != nil {
// 		log.Print(err)
// 	}

// 	for _, profile := range out.Mobileconfigs {
// 		if out.DeviceUDIDs != nil {
// 			// Not empty list
// 			if len(out.DeviceUDIDs) > 0 {
// 				// Shared profiles
// 				if out.DeviceUDIDs[0] == "*" {
// 					var devices = GetAllDevices()
// 					var deviceIds []string
// 					for _, item := range devices {
// 						deviceIds = append(deviceIds, item.UDID)
// 					}
// 					err := db.DB.Model(&sharedProfileModel).Where("payload_uuid = ? and payload_identifier = ?", profile.UUID, profile.PayloadIdentifier).Update("installed = ?", false).Update("installed", false).Scan(&sharedProfiles).Error
// 					if err != nil {
// 						log.Print(err)
// 						continue
// 					}

// 					DeleteSharedProfiles(devices, sharedProfiles)

// 				} else {
// 					var deviceIds []string
// 					for _, item := range out.DeviceUDIDs {
// 						device := GetDevice(item)
// 						devices = append(devices, device)
// 						deviceIds = append(deviceIds, device.UDID)
// 					}

// 					err := db.DB.Model(&profilesModel).Where("payload_uuid = ? and payload_identifier = ? and device_ud_id IN (?)", profile.UUID, profile.PayloadIdentifier, deviceIds).Update("installed", false).Scan(&profiles).Error
// 					if err != nil {
// 						log.Print(err)
// 						continue
// 					}

// 					DeleteDeviceProfiles(devices, profiles)
// 				}

// 			}
// 		}
// 	}
// }

func SaveInstallApplications(devices []types.Device, payload types.InstallApplicationPayload) {
	var installApplication types.DeviceInstallApplication

	for _, device := range devices {
		for _, ManifestURL := range payload.ManifestURLs {
			installApplication.ManifestURL = ManifestURL.URL
			installApplication.DeviceUDID = device.UDID
			err := db.DB.Model(&device).Where("device_ud_id = ? AND manifest_url = ?", device.UDID, ManifestURL.URL).Assign(&installApplication).FirstOrCreate(&installApplication).Error
			if err != nil {
				log.Print(err)
			}
		}
	}
}

func PushInstallApplication(devices []types.Device, installApplication types.DeviceInstallApplication) {
	for _, device := range devices {

		var commandPayload types.CommandPayload
		commandPayload.UDID = device.UDID
		commandPayload.RequestType = "InstallApplication"
		commandPayload.ManifestURL = installApplication.ManifestURL

		SendCommand(commandPayload)

	}

}

func SaveSharedInstallApplications(payload types.InstallApplicationPayload) {
	var sharedInstallApplication types.SharedInstallApplication
	if len(payload.ManifestURLs) == 0 {
		return
	}
	tx := db.DB.Model(&sharedInstallApplication)
	for _, ManifestURL := range payload.ManifestURLs {
		sharedInstallApplication.ManifestURL = ManifestURL.URL
		tx = tx.Assign(&sharedInstallApplication).FirstOrCreate(&sharedInstallApplication)
	}

	err := tx.Error
	if err != nil {
		fmt.Print(err)
	}
}

func PushSharedInstallApplication(devices []types.Device, installSharedApplication types.SharedInstallApplication) {
	for _, device := range devices {

		var commandPayload types.CommandPayload
		commandPayload.UDID = device.UDID
		commandPayload.RequestType = "InstallApplication"
		commandPayload.ManifestURL = installSharedApplication.ManifestURL

		SendCommand(commandPayload)

	}

}

func InstallBootstrapPackages(device types.Device) {
	var sharedInstallApplication types.SharedInstallApplication
	var deviceInstallApplication types.DeviceInstallApplication
	var sharedInstallApplications []types.SharedInstallApplication
	var deviceInstallApplications []types.DeviceInstallApplication
	var devices []types.Device

	devices = append(devices, device)

	err := db.DB.Model(&sharedInstallApplication).Scan(&sharedInstallApplications).Error
	if err != nil {
		log.Print(err)
	}

	// Push all the apps
	for _, savedApp := range sharedInstallApplications {
		PushSharedInstallApplication(devices, savedApp)
	}

	err = db.DB.Model(&deviceInstallApplication).Where("device_ud_id = ?", device.UDID).Scan(&deviceInstallApplications).Error
	if err != nil {
		log.Print(err)
	}

	// Push all the apps
	for _, savedApp := range deviceInstallApplications {
		PushInstallApplication(devices, savedApp)
	}
}

// func GetDeviceProfiles(w http.ResponseWriter, r *http.Request) {
// 	var profiles []types.DeviceProfile
// 	vars := mux.Vars(r)

// 	err := db.DB.Find(&profiles).Where("device_ud_id = ?", vars["udid"]).Scan(&profiles).Error
// 	if err != nil {
// 		fmt.Println(err)
// 		log.Print("Couldn't scan to Device model")
// 	}
// 	output, err := json.MarshalIndent(&profiles, "", "    ")
// 	if err != nil {
// 		log.Print(err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 	}

// 	w.Write(output)

// }
