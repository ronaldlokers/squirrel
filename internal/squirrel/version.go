// Package squirrel is the core: captures, durability, people and storage.
// It knows nothing about any chat system. Nothing here may import
// internal/transport — Go rejects the resulting import cycle, which is what
// enforces the boundary this design rests on.
package squirrel

// Name identifies the service in logs and in the user agent.
const Name = "squirrel"
