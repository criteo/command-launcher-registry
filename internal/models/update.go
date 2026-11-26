package models

// RegistryUpdate is the body of PUT /registry/{name}, a partial update.
//
// A nil field was absent from the body (or null) and keeps the stored
// value. A non-nil field replaces the stored value; an empty value ("", [],
// {}) clears it. Name is optional and only checked against the URL.
type RegistryUpdate struct {
	Name         *string            `json:"name"`
	Description  *string            `json:"description"`
	Admins       *[]string          `json:"admins"`
	CustomValues *map[string]string `json:"custom_values"`
}

// ApplyTo sets the fields present in u on r. It does not touch r.Name or
// r.Packages.
func (u *RegistryUpdate) ApplyTo(r *Registry) {
	if u.Description != nil {
		r.Description = *u.Description
	}
	if u.Admins != nil {
		r.Admins = *u.Admins
	}
	if u.CustomValues != nil {
		r.CustomValues = *u.CustomValues
	}
}

// PackageUpdate is the body of PUT /registry/{name}/package/{package}, a
// partial update with the same rules as RegistryUpdate.
type PackageUpdate struct {
	Name         *string            `json:"name"`
	Description  *string            `json:"description"`
	Maintainers  *[]string          `json:"maintainers"`
	CustomValues *map[string]string `json:"custom_values"`
}

// ApplyTo sets the fields present in u on p. It does not touch p.Name or
// p.Versions.
func (u *PackageUpdate) ApplyTo(p *Package) {
	if u.Description != nil {
		p.Description = *u.Description
	}
	if u.Maintainers != nil {
		p.Maintainers = *u.Maintainers
	}
	if u.CustomValues != nil {
		p.CustomValues = *u.CustomValues
	}
}

// Validate checks u as the update of the registry named name: Name, when
// present, must equal name, and the fields present must pass
// ValidateRegistry. Absent fields are not checked, so stored values are
// never rejected by an update that leaves them alone.
func (u *RegistryUpdate) Validate(name string) error {
	if u.Name != nil && *u.Name != name {
		return &ValidationError{Field: "name", Message: "name in body must match the registry name in the URL"}
	}
	r := Registry{Name: name}
	u.ApplyTo(&r)
	return ValidateRegistry(&r)
}

// Validate checks u as the update of the package named name, with the same
// rules as RegistryUpdate.Validate and ValidatePackage.
func (u *PackageUpdate) Validate(name string) error {
	if u.Name != nil && *u.Name != name {
		return &ValidationError{Field: "name", Message: "name in body must match the package name in the URL"}
	}
	p := Package{Name: name}
	u.ApplyTo(&p)
	return ValidatePackage(&p)
}
