package config

import "errors"

// GCEInstance wraps a CFT GCE Instance.
type GCEInstance struct {
	GCEInstanceProperties `json:"properties"`
	CustomBootImage       *struct {
		ImageName string `json:"image_name"`
	} `json:"custom_boot_image,omitempty"`
}

// GCEInstanceProperties represents a partial CFT instance implementation.
type GCEInstanceProperties struct {
	GCEInstanceName string `json:"name"`
	Zone            string `json:"zone"`
	DiskImage       string `json:"diskImage,omitempty"`
}

// Init initializes the instance.
func (i *GCEInstance) Init(*Project) error {
	if i.CustomBootImage != nil {
		if i.DiskImage != "" {
			return errors.New("custom boot image and disk image cannot both be set")
		}
		i.DiskImage = "global/images/" + i.CustomBootImage.ImageName
	}
	return nil
}

// Name returns the name of this instance.
func (i *GCEInstance) Name() string {
	return i.GCEInstanceName
}

// TemplatePath returns the name of the template to use for this instance.
func (i *GCEInstance) TemplatePath() string {
	return "deploy/config/templates/instance/instance.py"
}
