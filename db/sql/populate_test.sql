---
-- Some test content; do not use in production!
---
INSERT INTO location VALUES (1, 'kitchen');
INSERT INTO location VALUES (2, 'living');
INSERT INTO location VALUES (3, 'bedroom');

INSERT INTO device VALUES (1, 'upward', 'lamp_v1', '0001', 'kitchen overhead', 1);
INSERT INTO device VALUES (2, 'google', 'chromecast', 'Living Room', 'Living Room', 2);
INSERT INTO device VALUES (3, 'upward', 'lamp_v1', '0002', 'lamp1', 2);
INSERT INTO device VALUES (4, 'upward', 'lamp_v1', '0003', 'lamp2', 3);
INSERT INTO device VALUES (5, 'iLuv', 'Syren', 'sy1234b', 'bedroom speakers', 3);
INSERT INTO device VALUES (6, 'google', 'chromecast', 'Bedroom 1', 'Bedroom 1', 3);
INSERT INTO device VALUES (7, 'google', 'chromecast', 'Bedroom 2', 'Bedroom 2', 3);

INSERT INTO component VALUES (1, 1, 'bulb1', 'light_emitter');
INSERT INTO component VALUES (2, 2, 'bulb2', 'light_emitter');
INSERT INTO component VALUES (3, 2, 'chromecast_media_player', 'media_player');
INSERT INTO component VALUES (4, 3, 'bulb1', 'light_emitter');
INSERT INTO component VALUES (5, 4, 'bulb1', 'light_emitter');
INSERT INTO component VALUES (6, 4, 'bulb2', 'light_emitter');
INSERT INTO component VALUES (7, 5, 'speaker1', 'speaker');
INSERT INTO component VALUES (8, 5, 'speaker2', 'speaker');
INSERT INTO component VALUES (9, 6, 'chromecast_media_player', 'media_player');
INSERT INTO component VALUES (10, 7, 'chromecast_media_player', 'media_player');


INSERT INTO light_emitter VALUES (1, 11);
INSERT INTO light_emitter VALUES (2, 12);
INSERT INTO light_emitter VALUES (4, 14);
INSERT INTO light_emitter VALUES (5, 15);
INSERT INTO light_emitter VALUES (6, 16);

INSERT INTO media_player VALUES (3, 'STOPPED');
INSERT INTO media_player VALUES (9, 'PLAYING');
INSERT INTO media_player VALUES (10, 'STOPPED');