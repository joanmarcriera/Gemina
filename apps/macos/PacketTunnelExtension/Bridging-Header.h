// Objective-C bridging header for the Xcode build: exposes the Go transport
// core's C ABI to Swift (CoreTransport.swift) without a Swift module. SwiftPM
// uses the CGeminaCore module instead; both reach the same symbols, which
// are linked from the Go c-archive (build/libgeminacore.a).
#import "geminacore.h"
