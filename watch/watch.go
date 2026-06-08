// Package watch is a listener that constantly looks for new commit hashs
// for the source provided in the background.
//
// it is meant to be run in systemd service. 
// it automatically archives into zip if hash changed
package watch
