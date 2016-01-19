# SIFT types

## Devices and Components

SIFT divides it's representation of connected devices into two distinct
objects:

* Components: The functional elements of a connected device. Components can make
  noise, read temperatures, and play movies.
* Devices: A physical container of Components. Devices can be physically located
  and have network addresses.

Devices can have any number of Components, which are internally addressable by a
locally-unique name. Each Component is attached to exactly one Device. Complex
real-world connected devices can be represented as a single Device with many
Components.

## Component types

Each Component must have a type. The type of component dictates it's behavior.

LightEmitter is the SIFT Component Type representing a light source, like a
light bulb.

```go
type LightEmitter struct {
	BaseComponent // adds `Make, Model string` and some methods
	State LightEmitterState
	Stats *LightEmitterStats
	Specs *LightEmitterSpecs
}

type LightEmitterState struct {
	BrightnessInPercent uint8 `db:"brightness_in_percent" json:"brightness_in_percent"`
}

type LightEmitterStats struct {
	HoursOn int `db:"hours_on"`
}

type LightEmitterSpecs struct {
	MaxOutputInLumens       int `db:"max_output_in_lumens" json:"max_output_in_lumens"`
	MinOutputInLumens       int `db:"min_output_in_lumens" json:"min_output_in_lumens"`
	ExpectedLifetimeInHours int `db:"expected_lifetime_in_hours" json:"expected_lifetime_in_hours"`
}
```

Most Component Types will follow a similar structure, with three main parts:
* State: The active, changing, often-mutable qualities of the Component.
* Stats: Aggregated statistics about the specific component, produced by the
  SIFT Server
* Specs: Specifications specific to this make and model of Component. Note that
  all Components with the same make and model should share identical Specs.

## Intents

Intents generally describe a desire for something to happen. Each Intent must
have a type.

Most often, Intents describe a specific desired change for a specific Component.
Many Intents are specific to a particular Intent Type.

```go
type SetLightEmitterIntent struct {
	BrightnessInPercent uint8 `db:"brightness_in_percent" json:"brightness_in_percent"`
}
```
