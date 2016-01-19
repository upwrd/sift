CREATE TABLE IF NOT EXISTS location (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS device (
    id INTEGER PRIMARY KEY,
    manufacturer TEXT NOT NULL,
    external_id TEXT NOT NULL,
    name TEXT,
    location_id INTEGER,
    is_online INTEGER NOT NULL,
    FOREIGN KEY (location_id) REFERENCES location(id),
    CHECK(manufacturer <> ''),
    CHECK(external_id <> ''),
    CHECK(id <> 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS device_by_manufacturer_external_id
    ON device ( manufacturer, external_id );

CREATE TABLE IF NOT EXISTS component (
    id INTEGER PRIMARY KEY,
    device_id INTEGER,
    name TEXT NOT NULL,
    make TEXT NOT NULL,
    model TEXT NOT NULL,
    type TEXT NOT NULL,
    FOREIGN KEY (device_id) REFERENCES device(id),
    CHECK(name <> ''),
    CHECK(device_id <> 0),
    CHECK(type <> ''),
    UNIQUE(name, device_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS component_by_device_and_name
    ON component ( device_id, name );


--
-- light emitters
--

CREATE TABLE IF NOT EXISTS light_emitter_state (
    id INTEGER PRIMARY KEY,
    brightness_in_percent INTEGER,
    FOREIGN KEY (id) REFERENCES component(id),
    CHECK(id <> 0)
);

CREATE TABLE IF NOT EXISTS light_emitter_spec (
    make TEXT NOT NULL, -- Electro
    model TEXT NOT NULL, -- HydroFlex0.0.1
    max_output_in_lumens INTEGER,
    min_output_in_lumens INTEGER,
    expected_lifetime_in_hours INTEGER
);

CREATE UNIQUE INDEX IF NOT EXISTS light_emitter_spec_by_make_model
    ON light_emitter_spec ( make, model );

CREATE TABLE IF NOT EXISTS light_emitter_stats (
    id INTEGER PRIMARY KEY,
    hours_on INTEGER
);

--
-- media players
--

CREATE TABLE IF NOT EXISTS media_player_state (
    id INTEGER PRIMARY KEY,
    play_state TEXT,
    media_type TEXT,
    source TEXT,
    FOREIGN KEY (id) REFERENCES component(id),
    CHECK(id <> 0)
);

CREATE TABLE IF NOT EXISTS media_player_spec (
    make TEXT NOT NULL, -- Electro
    model TEXT NOT NULL, -- HydroFlex0.0.1
    supported_audio_types TEXT NOT NULL,
    supported_video_types TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS media_player_spec_by_make_model
    ON media_player_spec ( make, model );

CREATE TABLE IF NOT EXISTS media_player_stats (
    id INTEGER PRIMARY KEY,
    hours_on INTEGER
);