// Package piglogin is Ompig's built-in sprite login: the sprite's 16-by-14 ompig, which the startup header shows in Pi's logo
// slot, the full login art (the 12-pixel "Ompig." hero wordmark with its drop shadow beside that ompig) that /sprite preview shows,
// the sprites `/sprite` chooses from, and the selection kept under $OMPIG_HOME/state/ompig-standard/login.json.
//
// The artwork, sprite names and colors, the login definition and the golden renders of the color sprites and the sheriff
// are Ompig Standard's piglogin extension (ompiglets/standard/extensions/piglogin at tmih06/ompig d86eb93), written by
// Michael Kinsy and carried by Pigpen's ompig-login Package under the MIT license. The character sprites (PiGrogu, Darth
// Vader, Kratos, Ompiglet, Spider-Ham) are the owner's earlier Ompig piglogin. The pixel data is unchanged.
//
// ompig divergence (D2): Pi's startup header draws Pi's logo (pi-logo.ts, interactive-mode.ts:998-1006); Ompig draws the
// sprite's ompig.
package piglogin
