package sql

// PopulateSpecsSQL is a collection of sqlite statements used to populate the
// SIFT database with specs of known Components.
var PopulateSpecsSQL = `
INSERT INTO 'light_emitter_spec'
    ('make', 'model','max_output_in_lumens',
    'min_output_in_lumens', 'expected_lifetime_in_hours')
    VALUES
    ('example', 'light_emitter_1', 700, 0, 10000),
    ("connected_by_tcp", "bulb", 950, 0, 199728);
`
